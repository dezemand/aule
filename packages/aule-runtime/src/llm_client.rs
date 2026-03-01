use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use anyhow::{Context, Result, anyhow, bail};
use futures_util::StreamExt;
use log::warn;
use reqwest::Client;
use reqwest::StatusCode;
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};

const MAX_TOOL_CALLS_PER_RESPONSE: usize = 8;

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
    Invalid { error: String },
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
    /// The stream failed transiently and will be retried.
    Retry { attempt: u32, error: String },
    /// Generation is complete; final tool decision is available.
    Done,
}

#[derive(Debug)]
enum LlmStreamError {
    Retryable(anyhow::Error),
    Fatal(anyhow::Error),
}

impl LlmStreamError {
    fn retryable(err: anyhow::Error) -> Self {
        Self::Retryable(err)
    }

    fn fatal(err: anyhow::Error) -> Self {
        Self::Fatal(err)
    }
}

// ---------------------------------------------------------------------------
// Final tool decision returned after stream completes
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub struct ToolDecision {
    pub assistant_message: Value,
    pub tool_call_id: String,
    pub action: AgentAction,
    /// IDs of additional tool calls the model made in the same turn.
    /// The caller must provide tool results for these (OpenAI requires it).
    pub extra_tool_call_ids: Vec<String>,
}

// ---------------------------------------------------------------------------
// OpenAI client (streaming)
// ---------------------------------------------------------------------------

#[derive(Debug)]
pub struct OpenAiClient {
    api_key: String,
    model: String,
    max_retries: u32,
    retry_base_delay_ms: u64,
    retry_max_delay_ms: u64,
    http: Client,
    rt: tokio::runtime::Runtime,
}

impl OpenAiClient {
    pub fn new(
        api_key: String,
        model: String,
        max_retries: u32,
        retry_base_delay_ms: u64,
        retry_max_delay_ms: u64,
        rt: tokio::runtime::Runtime,
    ) -> Result<Self> {
        let http = Client::builder()
            .connect_timeout(Duration::from_secs(30))
            .read_timeout(Duration::from_secs(120))
            .build()
            .context("Failed to build HTTP client")?;
        Ok(Self {
            api_key,
            model,
            max_retries,
            retry_base_delay_ms,
            retry_max_delay_ms,
            http,
            rt,
        })
    }

    /// Stream a tool decision from OpenAI. Calls `on_event` with buffered
    /// deltas (~250 ms) so the caller can persist live runtime logs without
    /// flooding SpacetimeDB.
    pub fn stream_tool_decision(
        &self,
        conversation: &Conversation,
        mut on_event: impl FnMut(StreamEvent),
    ) -> Result<ToolDecision> {
        for attempt in 0..=self.max_retries {
            match self
                .rt
                .block_on(self.stream_inner(conversation, &mut on_event))
            {
                Ok(decision) => return Ok(decision),
                Err(LlmStreamError::Fatal(err)) => return Err(err),
                Err(LlmStreamError::Retryable(err)) => {
                    if attempt >= self.max_retries {
                        return Err(err.context(format!(
                            "OpenAI streaming failed after {} attempts",
                            self.max_retries + 1
                        )));
                    }

                    let delay_ms = compute_retry_delay_ms(
                        self.retry_base_delay_ms,
                        self.retry_max_delay_ms,
                        attempt,
                    );
                    let next_attempt = attempt + 2;
                    warn!(
                        "OpenAI streaming transient failure (attempt {}/{}): {:#}; retrying in {}ms",
                        attempt + 1,
                        self.max_retries + 1,
                        err,
                        delay_ms
                    );
                    on_event(StreamEvent::Retry {
                        attempt: next_attempt,
                        error: format!("{err:#}"),
                    });
                    std::thread::sleep(Duration::from_millis(delay_ms));
                }
            }
        }

        Err(anyhow!("OpenAI streaming retry loop ended unexpectedly"))
    }

    async fn stream_inner(
        &self,
        conversation: &Conversation,
        on_event: &mut dyn FnMut(StreamEvent),
    ) -> std::result::Result<ToolDecision, LlmStreamError> {
        let body = json!({
            "model": self.model,
            "temperature": 0.1,
            "stream": true,
            "messages": conversation.messages(),
            "tools": tool_definitions(),
            "tool_choice": "required",
            "parallel_tool_calls": false,
        });

        let response = self
            .http
            .post("https://api.openai.com/v1/chat/completions")
            .bearer_auth(&self.api_key)
            .json(&body)
            .send()
            .await
            .map_err(|err| {
                LlmStreamError::retryable(anyhow!("OpenAI streaming request failed: {err:#}"))
            })?;

        let status = response.status();
        if !status.is_success() {
            let err_body = response
                .text()
                .await
                .unwrap_or_else(|_| "<failed to read body>".to_string());
            let err = anyhow!("OpenAI request failed with {status}: {err_body}");
            if is_retryable_http_status(status) {
                return Err(LlmStreamError::retryable(err));
            }
            return Err(LlmStreamError::fatal(err));
        }

        // Accumulation state — track tool calls by index so parallel calls
        // from the model don't get their arguments concatenated together.
        let mut content_buf = String::new();
        let mut tool_calls: Vec<(String, String, String)> = Vec::new(); // (id, name, args)

        // Buffered flush state
        let mut pending_text = String::new();
        let mut pending_args = String::new();
        let mut active_tool_name = String::new();
        let mut last_flush = Instant::now();
        let flush_interval = Duration::from_millis(250);

        // Consume the response as a raw byte stream, buffering bytes until
        // complete lines are available. This avoids corrupting multibyte
        // UTF-8 characters that may be split across HTTP chunks.
        let mut byte_stream = response.bytes_stream();
        let mut line_buf: Vec<u8> = Vec::new();
        let mut done = false;

        let mut process_sse_line = |raw_line: &str| -> std::result::Result<bool, LlmStreamError> {
            let line = raw_line.trim_end();
            if !line.starts_with("data: ") {
                return Ok(false);
            }

            let data = &line["data: ".len()..];
            if data == "[DONE]" {
                return Ok(true);
            }

            let chunk: SseChunk = match serde_json::from_str(data) {
                Ok(c) => c,
                Err(e) => {
                    log::debug!("Skipping malformed SSE chunk: {e} — data: {data}");
                    return Ok(false);
                }
            };

            let Some(choice) = chunk.choices.first() else {
                return Ok(false);
            };
            let delta = &choice.delta;

            // Content delta
            if let Some(ref text_delta) = delta.content {
                content_buf.push_str(text_delta);
                pending_text.push_str(text_delta);
            }

            // Tool call deltas — tracked by index
            if let Some(ref tc_deltas) = delta.tool_calls {
                for tc_delta in tc_deltas {
                    let idx = tc_delta.index.unwrap_or(0);

                    if idx >= MAX_TOOL_CALLS_PER_RESPONSE {
                        return Err(LlmStreamError::fatal(anyhow!(
                            "OpenAI stream returned tool call index {} exceeding maximum {}",
                            idx,
                            MAX_TOOL_CALLS_PER_RESPONSE
                        )));
                    }

                    if idx > tool_calls.len() {
                        return Err(LlmStreamError::fatal(anyhow!(
                            "OpenAI stream returned out-of-order tool call index {} (current_len={})",
                            idx,
                            tool_calls.len()
                        )));
                    }

                    if idx == tool_calls.len() {
                        tool_calls.push((String::new(), String::new(), String::new()));
                    }

                    let entry = &mut tool_calls[idx];

                    if let Some(ref id) = tc_delta.id {
                        entry.0 = id.clone();
                    }
                    if let Some(ref func) = tc_delta.function {
                        if let Some(ref name) = func.name {
                            entry.1 = name.clone();
                            active_tool_name = name.clone();
                        }
                        if let Some(ref args) = func.arguments {
                            entry.2.push_str(args);
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
                        tool_name: active_tool_name.clone(),
                        args_delta: std::mem::take(&mut pending_args),
                    });
                }
                last_flush = Instant::now();
            }

            Ok(false)
        };

        while let Some(chunk_result) = byte_stream.next().await {
            let chunk_bytes = chunk_result.map_err(|err| {
                LlmStreamError::retryable(anyhow!("Error reading SSE stream chunk: {err:#}"))
            })?;
            line_buf.extend_from_slice(&chunk_bytes);

            // Process all complete lines in the buffer
            while let Some(newline_pos) = line_buf.iter().position(|&b| b == b'\n') {
                let line_bytes: Vec<u8> = line_buf.drain(..=newline_pos).collect();
                let line = String::from_utf8_lossy(&line_bytes);
                done = process_sse_line(&line)?;
                if done {
                    break;
                }
            }

            if done {
                break;
            }
        }

        if !done && !line_buf.is_empty() {
            let line = String::from_utf8_lossy(&line_buf);
            done = process_sse_line(&line)?;
            line_buf.clear();
        }

        // Final flush of any remaining buffered content
        if !pending_text.is_empty() {
            on_event(StreamEvent::TextDelta(pending_text));
        }
        if !pending_args.is_empty() {
            on_event(StreamEvent::ToolArgsDelta {
                tool_name: active_tool_name,
                args_delta: pending_args,
            });
        }

        // Verify the stream completed normally (received `data: [DONE]`)
        // before attempting to parse the tool call.
        if !done {
            return Err(LlmStreamError::retryable(anyhow!(
                "OpenAI SSE stream ended without a [DONE] marker (connection may have dropped)"
            )));
        }

        if tool_calls.is_empty() {
            return Err(LlmStreamError::fatal(anyhow!(
                "OpenAI stream completed without a tool call"
            )));
        }

        // Use the first tool call as the action to execute this turn.
        let (ref tc_id, ref tc_name, ref tc_args) = tool_calls[0];
        if tc_id.is_empty() || tc_name.is_empty() {
            return Err(LlmStreamError::fatal(anyhow!(
                "OpenAI stream completed with an incomplete tool call"
            )));
        }

        let action = match action_from_tool_call(tc_name, tc_args) {
            Ok(action) => action,
            Err(err) => AgentAction::Invalid {
                error: format!(
                    "Invalid tool call '{}' (id={}): {:#}. Return a valid tool call.",
                    tc_name, tc_id, err
                ),
            },
        };
        on_event(StreamEvent::Done);

        // Build the assistant message including ALL tool calls for correct
        // conversation state (OpenAI expects a tool result for each).
        let tool_calls_json: Vec<Value> = tool_calls
            .iter()
            .map(|(id, name, args)| {
                json!({
                    "id": id,
                    "type": "function",
                    "function": {
                        "name": name,
                        "arguments": args,
                    }
                })
            })
            .collect();

        let assistant_message = json!({
            "role": "assistant",
            "content": if content_buf.is_empty() { Value::Null } else { Value::String(content_buf) },
            "tool_calls": tool_calls_json,
        });

        // Collect the extra tool call IDs so the caller can provide
        // placeholder tool results for them (required by OpenAI).
        let extra_tool_call_ids: Vec<String> = tool_calls[1..]
            .iter()
            .filter_map(|(id, _, _)| {
                let id = id.trim();
                if id.is_empty() {
                    None
                } else {
                    Some(id.to_string())
                }
            })
            .collect();

        Ok(ToolDecision {
            assistant_message,
            tool_call_id: tc_id.clone(),
            action,
            extra_tool_call_ids,
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

fn is_retryable_http_status(status: StatusCode) -> bool {
    matches!(
        status,
        StatusCode::TOO_MANY_REQUESTS
            | StatusCode::INTERNAL_SERVER_ERROR
            | StatusCode::BAD_GATEWAY
            | StatusCode::SERVICE_UNAVAILABLE
            | StatusCode::GATEWAY_TIMEOUT
    )
}

fn compute_retry_delay_ms(base_ms: u64, max_ms: u64, attempt: u32) -> u64 {
    let exp = 1u64.checked_shl(attempt.min(16)).unwrap_or(u64::MAX);
    let backoff = base_ms.saturating_mul(exp).min(max_ms);
    let jitter_range = (base_ms / 2).max(1);
    let jitter = pseudo_random_u64() % jitter_range;
    backoff.saturating_add(jitter).min(max_ms)
}

fn pseudo_random_u64() -> u64 {
    let nanos = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_nanos() as u64)
        .unwrap_or(0);
    nanos.rotate_left(13) ^ nanos.wrapping_mul(0x9E37_79B9_7F4A_7C15)
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
    index: Option<usize>,
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
    use reqwest::StatusCode;

    use super::{
        AgentAction, Conversation, ObserveKind, action_from_tool_call, compute_retry_delay_ms,
        is_retryable_http_status,
    };

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
        let err = action_from_tool_call("aule_observe", r#"{"kind":"bogus","content":"hi"}"#)
            .unwrap_err();
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

    #[test]
    fn retryable_http_statuses_are_classified_correctly() {
        assert!(is_retryable_http_status(StatusCode::TOO_MANY_REQUESTS));
        assert!(is_retryable_http_status(StatusCode::INTERNAL_SERVER_ERROR));
        assert!(is_retryable_http_status(StatusCode::BAD_GATEWAY));
        assert!(is_retryable_http_status(StatusCode::SERVICE_UNAVAILABLE));
        assert!(is_retryable_http_status(StatusCode::GATEWAY_TIMEOUT));

        assert!(!is_retryable_http_status(StatusCode::BAD_REQUEST));
        assert!(!is_retryable_http_status(StatusCode::UNAUTHORIZED));
        assert!(!is_retryable_http_status(StatusCode::FORBIDDEN));
    }

    #[test]
    fn retry_delay_respects_bounds() {
        let base_ms = 1_000;
        let max_ms = 10_000;
        let delay_attempt_0 = compute_retry_delay_ms(base_ms, max_ms, 0);
        assert!((1_000..=1_499).contains(&delay_attempt_0));

        let delay_large_attempt = compute_retry_delay_ms(base_ms, max_ms, 8);
        assert!(delay_large_attempt <= max_ms);
        assert!(delay_large_attempt > 0);
    }
}
