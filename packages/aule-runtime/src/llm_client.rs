use std::time::{Duration, Instant};

use anyhow::{Context, Result, bail};
use futures_util::StreamExt;
use reqwest::Client;
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};

// ---------------------------------------------------------------------------
// Conversation state
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub struct Conversation {
    messages: Vec<Value>,
}

impl Conversation {
    pub fn new(system_prompt: &str, user_prompt: &str) -> Self {
        Self {
            messages: vec![
                json!({ "role": "system", "content": system_prompt }),
                json!({ "role": "user", "content": user_prompt }),
            ],
        }
    }

    pub fn messages(&self) -> &[Value] {
        &self.messages
    }

    pub fn push_assistant_message(&mut self, assistant_message: Value) {
        self.messages.push(assistant_message);
    }

    pub fn push_tool_result(&mut self, tool_call_id: &str, content: String) {
        self.messages.push(json!({
            "role": "tool",
            "tool_call_id": tool_call_id,
            "content": content,
        }));
    }
}

// ---------------------------------------------------------------------------
// Agent actions (parsed from tool calls)
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum ObserveKind {
    Progress,
    Finding,
    Error,
    Result,
}

impl ObserveKind {
    fn parse(s: &str) -> Option<Self> {
        match s {
            "progress" => Some(Self::Progress),
            "finding" => Some(Self::Finding),
            "error" => Some(Self::Error),
            "result" => Some(Self::Result),
            _ => None,
        }
    }

    pub fn as_str(&self) -> &'static str {
        match self {
            Self::Progress => "progress",
            Self::Finding => "finding",
            Self::Error => "error",
            Self::Result => "result",
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "action", rename_all = "snake_case")]
pub enum AgentAction {
    Shell { command: String },
    Observe { kind: ObserveKind, content: String },
    Status { content: String },
    Finish { result: String },
    Fail { error: String },
}

// ---------------------------------------------------------------------------
// Stream events emitted to the caller during generation
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub enum StreamEvent {
    /// Incremental text the model is producing (content delta).
    TextDelta(String),
    /// The model is building tool-call arguments (partial JSON).
    ToolArgsDelta {
        tool_name: String,
        args_delta: String,
    },
    /// Generation is complete; final tool decision is available.
    Done,
}

// ---------------------------------------------------------------------------
// Final tool decision returned after stream completes
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub struct ToolDecision {
    pub assistant_message: Value,
    pub tool_call_id: String,
    pub action: AgentAction,
}

// ---------------------------------------------------------------------------
// OpenAI client (streaming)
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub struct OpenAiClient {
    api_key: String,
    model: String,
    http: Client,
    rt: tokio::runtime::Handle,
}

impl OpenAiClient {
    pub fn new(api_key: String, model: String, rt: tokio::runtime::Handle) -> Result<Self> {
        let http = Client::builder()
            .connect_timeout(Duration::from_secs(30))
            .read_timeout(Duration::from_secs(120))
            .build()
            .context("Failed to build HTTP client")?;
        Ok(Self {
            api_key,
            model,
            http,
            rt,
        })
    }

    /// Stream a tool decision from OpenAI. Calls `on_event` with buffered
    /// deltas (~250 ms) so the caller can emit live observations without
    /// flooding SpacetimeDB.
    pub fn stream_tool_decision(
        &self,
        conversation: &Conversation,
        mut on_event: impl FnMut(StreamEvent),
    ) -> Result<ToolDecision> {
        self.rt
            .block_on(self.stream_inner(conversation, &mut on_event))
    }

    async fn stream_inner(
        &self,
        conversation: &Conversation,
        on_event: &mut dyn FnMut(StreamEvent),
    ) -> Result<ToolDecision> {
        let body = json!({
            "model": self.model,
            "temperature": 0.1,
            "stream": true,
            "messages": conversation.messages(),
            "tools": tool_definitions(),
            "tool_choice": "required",
        });

        let response = self
            .http
            .post("https://api.openai.com/v1/chat/completions")
            .bearer_auth(&self.api_key)
            .json(&body)
            .send()
            .await
            .context("OpenAI streaming request failed")?;

        let status = response.status();
        if !status.is_success() {
            let err_body = response
                .text()
                .await
                .unwrap_or_else(|_| "<failed to read body>".to_string());
            bail!("OpenAI request failed with {status}: {err_body}");
        }

        // Accumulation state
        let mut content_buf = String::new();
        let mut tool_call_id = String::new();
        let mut tool_name = String::new();
        let mut tool_args = String::new();

        // Buffered flush state
        let mut pending_text = String::new();
        let mut pending_args = String::new();
        let mut last_flush = Instant::now();
        let flush_interval = Duration::from_millis(250);

        // Consume the response as a raw byte stream, buffering bytes until
        // complete lines are available. This avoids corrupting multibyte
        // UTF-8 characters that may be split across HTTP chunks.
        let mut byte_stream = response.bytes_stream();
        let mut line_buf: Vec<u8> = Vec::new();
        let mut done = false;

        while let Some(chunk_result) = byte_stream.next().await {
            let chunk_bytes = chunk_result.context("Error reading SSE stream chunk")?;
            line_buf.extend_from_slice(&chunk_bytes);

            // Process all complete lines in the buffer
            while let Some(newline_pos) = line_buf.iter().position(|&b| b == b'\n') {
                let line_bytes: Vec<u8> = line_buf.drain(..=newline_pos).collect();
                let line = String::from_utf8_lossy(&line_bytes);
                let line = line.trim_end();

                if !line.starts_with("data: ") {
                    continue;
                }
                let data = &line["data: ".len()..];
                if data == "[DONE]" {
                    done = true;
                    break;
                }

                let chunk: SseChunk = match serde_json::from_str(data) {
                    Ok(c) => c,
                    Err(e) => {
                        log::debug!("Skipping malformed SSE chunk: {e} — data: {data}");
                        continue;
                    }
                };

                let Some(choice) = chunk.choices.first() else {
                    continue;
                };
                let delta = &choice.delta;

                // Content delta
                if let Some(ref text_delta) = delta.content {
                    content_buf.push_str(text_delta);
                    pending_text.push_str(text_delta);
                }

                // Tool call deltas
                if let Some(ref tool_calls) = delta.tool_calls {
                    for tc_delta in tool_calls {
                        if let Some(ref id) = tc_delta.id {
                            tool_call_id = id.clone();
                        }
                        if let Some(ref func) = tc_delta.function {
                            if let Some(ref name) = func.name {
                                tool_name = name.clone();
                            }
                            if let Some(ref args) = func.arguments {
                                tool_args.push_str(args);
                                pending_args.push_str(args);
                            }
                        }
                    }
                }

                // Flush buffered deltas if interval elapsed
                if last_flush.elapsed() >= flush_interval {
                    if !pending_text.is_empty() {
                        on_event(StreamEvent::TextDelta(std::mem::take(&mut pending_text)));
                    }
                    if !pending_args.is_empty() {
                        on_event(StreamEvent::ToolArgsDelta {
                            tool_name: tool_name.clone(),
                            args_delta: std::mem::take(&mut pending_args),
                        });
                    }
                    last_flush = Instant::now();
                }
            }

            if done {
                break;
            }
        }

        // Final flush of any remaining buffered content
        if !pending_text.is_empty() {
            on_event(StreamEvent::TextDelta(pending_text));
        }
        if !pending_args.is_empty() {
            on_event(StreamEvent::ToolArgsDelta {
                tool_name: tool_name.clone(),
                args_delta: pending_args,
            });
        }

        // Verify the stream completed normally (received `data: [DONE]`)
        // before attempting to parse the tool call.
        if !done {
            bail!("OpenAI SSE stream ended without a [DONE] marker (connection may have dropped)");
        }

        if tool_call_id.is_empty() || tool_name.is_empty() {
            bail!("OpenAI stream completed without a tool call");
        }

        let action = action_from_tool_call(&tool_name, &tool_args)?;
        on_event(StreamEvent::Done);

        // Build the assistant message for conversation state
        let assistant_message = json!({
            "role": "assistant",
            "content": if content_buf.is_empty() { Value::Null } else { Value::String(content_buf) },
            "tool_calls": [{
                "id": tool_call_id,
                "type": "function",
                "function": {
                    "name": tool_name,
                    "arguments": tool_args,
                }
            }]
        });

        Ok(ToolDecision {
            assistant_message,
            tool_call_id,
            action,
        })
    }
}

// ---------------------------------------------------------------------------
// Tool call -> AgentAction parsing
// ---------------------------------------------------------------------------

pub fn action_from_tool_call(name: &str, arguments: &str) -> Result<AgentAction> {
    match name {
        "sh" => {
            let args: ShellArgs = serde_json::from_str(arguments)
                .with_context(|| format!("Invalid sh tool args: {arguments}"))?;
            if args.command.trim().is_empty() {
                bail!("sh.command must not be empty");
            }
            Ok(AgentAction::Shell {
                command: args.command,
            })
        }
        "aule_observe" => {
            let args: ObserveArgs = serde_json::from_str(arguments)
                .with_context(|| format!("Invalid aule_observe tool args: {arguments}"))?;
            if args.content.trim().is_empty() {
                bail!("aule_observe.content must not be empty");
            }
            let kind = ObserveKind::parse(&args.kind).with_context(|| {
                format!(
                    "Invalid aule_observe kind '{}': must be progress, finding, error, or result",
                    args.kind
                )
            })?;
            Ok(AgentAction::Observe {
                kind,
                content: args.content,
            })
        }
        "aule_status" => {
            let args: StatusArgs = serde_json::from_str(arguments)
                .with_context(|| format!("Invalid aule_status tool args: {arguments}"))?;
            if args.content.trim().is_empty() {
                bail!("aule_status.content must not be empty");
            }
            Ok(AgentAction::Status {
                content: args.content,
            })
        }
        "aule_finish" => {
            let args: FinishArgs = serde_json::from_str(arguments)
                .with_context(|| format!("Invalid aule_finish tool args: {arguments}"))?;
            if args.result.trim().is_empty() {
                bail!("aule_finish.result must not be empty");
            }
            Ok(AgentAction::Finish {
                result: args.result,
            })
        }
        "aule_fail" => {
            let args: FailArgs = serde_json::from_str(arguments)
                .with_context(|| format!("Invalid aule_fail tool args: {arguments}"))?;
            if args.error.trim().is_empty() {
                bail!("aule_fail.error must not be empty");
            }
            Ok(AgentAction::Fail { error: args.error })
        }
        other => bail!("Unknown tool call: {other}"),
    }
}

// ---------------------------------------------------------------------------
// Tool definitions sent to OpenAI
// ---------------------------------------------------------------------------

fn tool_definitions() -> Vec<Value> {
    vec![
        json!({
            "type": "function",
            "function": {
                "name": "sh",
                "description": "Run a shell command in the local workspace.",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "command": { "type": "string" }
                    },
                    "required": ["command"],
                    "additionalProperties": false
                }
            }
        }),
        json!({
            "type": "function",
            "function": {
                "name": "aule_observe",
                "description": "Post an observation. kind must be progress, finding, error, or result.",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "kind": {
                            "type": "string",
                            "enum": ["progress", "finding", "error", "result"]
                        },
                        "content": { "type": "string" }
                    },
                    "required": ["kind", "content"],
                    "additionalProperties": false
                }
            }
        }),
        json!({
            "type": "function",
            "function": {
                "name": "aule_status",
                "description": "Report current progress status.",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "content": { "type": "string" }
                    },
                    "required": ["content"],
                    "additionalProperties": false
                }
            }
        }),
        json!({
            "type": "function",
            "function": {
                "name": "aule_finish",
                "description": "Mark task as complete with final result.",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "result": { "type": "string" }
                    },
                    "required": ["result"],
                    "additionalProperties": false
                }
            }
        }),
        json!({
            "type": "function",
            "function": {
                "name": "aule_fail",
                "description": "Mark task as failed with error summary.",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "error": { "type": "string" }
                    },
                    "required": ["error"],
                    "additionalProperties": false
                }
            }
        }),
    ]
}

// ---------------------------------------------------------------------------
// SSE chunk types (OpenAI streaming format)
// ---------------------------------------------------------------------------

#[derive(Debug, Deserialize)]
struct SseChunk {
    choices: Vec<SseChoice>,
}

#[derive(Debug, Deserialize)]
struct SseChoice {
    delta: SseDelta,
}

#[derive(Debug, Deserialize)]
struct SseDelta {
    content: Option<String>,
    tool_calls: Option<Vec<SseToolCallDelta>>,
}

#[derive(Debug, Deserialize)]
struct SseToolCallDelta {
    id: Option<String>,
    function: Option<SseFunctionDelta>,
}

#[derive(Debug, Deserialize)]
struct SseFunctionDelta {
    name: Option<String>,
    arguments: Option<String>,
}

// ---------------------------------------------------------------------------
// Tool argument types
// ---------------------------------------------------------------------------

#[derive(Debug, Deserialize)]
struct ShellArgs {
    command: String,
}

#[derive(Debug, Deserialize)]
struct ObserveArgs {
    kind: String,
    content: String,
}

#[derive(Debug, Deserialize)]
struct StatusArgs {
    content: String,
}

#[derive(Debug, Deserialize)]
struct FinishArgs {
    result: String,
}

#[derive(Debug, Deserialize)]
struct FailArgs {
    error: String,
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::{AgentAction, Conversation, ObserveKind, action_from_tool_call};

    #[test]
    fn parses_sh_tool_call() {
        let action = action_from_tool_call("sh", r#"{"command":"ls"}"#).unwrap();
        match action {
            AgentAction::Shell { command } => assert_eq!(command, "ls"),
            _ => panic!("expected shell action"),
        }
    }

    #[test]
    fn parses_aule_observe_tool_call() {
        let action =
            action_from_tool_call("aule_observe", r#"{"kind":"finding","content":"found it"}"#)
                .unwrap();
        match action {
            AgentAction::Observe { kind, content } => {
                assert_eq!(kind, ObserveKind::Finding);
                assert_eq!(content, "found it");
            }
            _ => panic!("expected observe action"),
        }
    }

    #[test]
    fn errors_on_invalid_observe_kind() {
        let err =
            action_from_tool_call("aule_observe", r#"{"kind":"bogus","content":"hi"}"#).unwrap_err();
        assert!(err.to_string().contains("Invalid aule_observe kind"));
    }

    #[test]
    fn parses_aule_finish_tool_call() {
        let action = action_from_tool_call("aule_finish", r#"{"result":"done"}"#).unwrap();
        match action {
            AgentAction::Finish { result } => assert_eq!(result, "done"),
            _ => panic!("expected finish action"),
        }
    }

    #[test]
    fn errors_on_unknown_tool() {
        let err = action_from_tool_call("unknown", "{}").unwrap_err();
        assert!(err.to_string().contains("Unknown tool call"));
    }

    #[test]
    fn errors_on_empty_shell_command() {
        let err = action_from_tool_call("sh", r#"{"command":" "}"#).unwrap_err();
        assert!(err.to_string().contains("must not be empty"));
    }

    #[test]
    fn conversation_starts_with_two_messages() {
        let convo = Conversation::new("sys", "user");
        assert_eq!(convo.messages().len(), 2);
    }

    #[test]
    fn conversation_grows_with_tool_round_trip() {
        let mut convo = Conversation::new("sys", "user");
        convo.push_assistant_message(
            serde_json::json!({"role": "assistant", "content": "thinking"}),
        );
        convo.push_tool_result("call_123", "result text".to_string());
        assert_eq!(convo.messages().len(), 4);
        assert_eq!(convo.messages()[3]["tool_call_id"], "call_123");
    }
}
