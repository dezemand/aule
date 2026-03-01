import { useEffect, useMemo, useRef, useState } from "react";
import type { RuntimeEvent } from "../module_bindings/types";
import { Markdown } from "./Markdown";

// ── Timeline item types ─────────────────────────────────────────────

type TimelineItem =
  | {
      id: string;
      kind: "assistant";
      content: string;
      timestamp: string;
    }
  | {
      id: string;
      kind: "tool_use";
      title: string;
      action: string;
      shellOutput: string | null;
      result: string;
      timestamp: string;
    };

// ── Helpers ─────────────────────────────────────────────────────────

function parseKeyValue(content: string): Record<string, string> {
  const parsed: Record<string, string> = {};
  for (const line of content.split("\n")) {
    const i = line.indexOf("=");
    if (i <= 0) continue;
    const key = line.slice(0, i).trim();
    const value = line.slice(i + 1).trim();
    if (key) parsed[key] = value;
  }
  return parsed;
}

function stripToolArgLines(content: string): string {
  return content
    .replace(/^\s*\[tool_args:[^\]]+\].*$/gm, "")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}

function extractShellCommand(action: string): string | null {
  const match = action.match(/Shell\s*\{\s*command:\s*"([\s\S]*?)"\s*\}/);
  return match?.[1] ?? null;
}

function summarizeAction(action: string | undefined): string {
  if (!action) return "Tool called";
  const cmd = extractShellCommand(action);
  if (cmd) {
    const short = cmd.length > 80 ? `${cmd.slice(0, 77)}...` : cmd;
    return `Ran \`${short}\``;
  }
  if (action.startsWith("Finish")) return "Finished task";
  if (action.startsWith("Fail")) return "Failed task";
  if (action.startsWith("Observe")) return "Posted observation";
  if (action.startsWith("Status")) return "Reported status";
  const match = action.match(/^([A-Za-z_]+)\s*\{/);
  return match?.[1] ? `Called ${match[1]}` : "Tool called";
}

function summarizeResult(result: string | undefined): string | null {
  if (!result) return null;
  if (result.includes("task completed")) return "Task completed";
  if (result.includes("task failed")) return "Task failed";
  if (result.includes("shell_error")) return "Shell error";
  if (result.startsWith("exit_code=")) {
    const exitMatch = result.match(/exit_code=Some\((\d+)\)/);
    const timeMatch = result.match(/duration_ms=(\d+)/);
    const code = exitMatch?.[1] ?? "?";
    const ms = timeMatch?.[1];
    return ms ? `Exit ${code} (${ms}ms)` : `Exit ${code}`;
  }
  if (result.length > 100) return `${result.slice(0, 97)}...`;
  return result;
}

function countLines(text: string): number {
  return text ? text.split("\n").length : 0;
}

// ── Build timeline ──────────────────────────────────────────────────

interface TurnBucket {
  turn: number;
  toolCall: RuntimeEvent | null;
  toolResults: RuntimeEvent[];
  shellOutput: RuntimeEvent | null;
}

function buildTimeline(events: RuntimeEvent[]): TimelineItem[] {
  const items: TimelineItem[] = [];

  // Group non-LLM events by turn
  const turnMap = new Map<number, TurnBucket>();

  for (const event of events) {
    const tag = event.eventType.tag;

    if (tag === "LlmResponse") {
      const content = stripToolArgLines(event.content || "");
      if (content) {
        items.push({
          id: event.id,
          kind: "assistant",
          content,
          timestamp: event.updatedAt.toDate().toLocaleTimeString(),
        });
      }
      continue;
    }

    const turn = Number(event.turn);
    let bucket = turnMap.get(turn);
    if (!bucket) {
      bucket = { turn, toolCall: null, toolResults: [], shellOutput: null };
      turnMap.set(turn, bucket);
    }

    if (tag === "ToolCall") {
      bucket.toolCall = event;
    } else if (tag === "ToolResult") {
      bucket.toolResults.push(event);
    } else if (tag === "ShellOutput") {
      bucket.shellOutput = event;
    }
  }

  // Now interleave: walk events in order, emit assistant items immediately,
  // emit tool_use once per turn (on the first non-LLM event of that turn)
  const emittedTurns = new Set<number>();

  for (const event of events) {
    if (event.eventType.tag === "LlmResponse") continue;

    const turn = Number(event.turn);
    if (emittedTurns.has(turn)) continue;
    emittedTurns.add(turn);

    const bucket = turnMap.get(turn);
    if (!bucket) continue;

    const callParsed = bucket.toolCall
      ? parseKeyValue(bucket.toolCall.content)
      : {};
    const action = callParsed.action;

    // Pick the "main" tool result (skip extras / placeholders)
    const mainResult = bucket.toolResults.find((r) => {
      const p = parseKeyValue(r.content);
      return !p.result?.includes("Skipped:");
    });
    const resultParsed = mainResult ? parseKeyValue(mainResult.content) : {};

    const shellContent = bucket.shellOutput?.content ?? null;

    // Build combined details for the result section
    const resultSummary = summarizeResult(resultParsed.result);
    const resultRaw = mainResult?.content ?? "";

    items.push({
      id: bucket.toolCall?.id ?? `turn-${turn}`,
      kind: "tool_use",
      title: summarizeAction(action),
      action: action ?? "",
      shellOutput: shellContent,
      result: resultSummary ?? resultRaw,
      timestamp: (
        bucket.toolCall ?? mainResult ?? bucket.shellOutput
      )!.updatedAt
        .toDate()
        .toLocaleTimeString(),
    });
  }

  // Sort by original event order: items were added in event order for
  // assistant messages, but tool_use items need interleaving. Rebuild
  // in correct order by walking events once more.
  const itemById = new Map(items.map((it) => [it.id, it]));
  const ordered: TimelineItem[] = [];
  const seen = new Set<string>();

  for (const event of events) {
    const tag = event.eventType.tag;
    let id: string;

    if (tag === "LlmResponse") {
      id = event.id;
    } else {
      const turn = Number(event.turn);
      id = turnMap.get(turn)?.toolCall?.id ?? `turn-${turn}`;
    }

    if (seen.has(id)) continue;
    seen.add(id);

    const item = itemById.get(id);
    if (item) ordered.push(item);
  }

  return ordered;
}

// ── Component ───────────────────────────────────────────────────────

interface ConversationViewProps {
  events: RuntimeEvent[];
  taskDescription: string;
  taskStatus: string;
  systemPrompt?: string;
}

export function ConversationView({
  events,
  taskDescription,
  taskStatus,
  systemPrompt,
}: ConversationViewProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);

  const timeline = useMemo(() => buildTimeline(events), [events]);

  const lastEventUpdate =
    events.length > 0
      ? (events[events.length - 1]?.updatedAt.toDate().getTime() ?? 0)
      : 0;

  useEffect(() => {
    if (taskStatus !== "Running" || !autoScroll || !scrollRef.current) return;
    scrollRef.current.scrollTo({
      top: scrollRef.current.scrollHeight,
      behavior: "smooth",
    });
  }, [taskStatus, autoScroll, events.length, lastEventUpdate]);

  return (
    <div className="flex flex-col gap-3">
      {systemPrompt && (
        <details className="rounded-lg border border-gray-800 bg-gray-900/60">
          <summary className="cursor-pointer px-4 py-2 text-xs text-gray-500 hover:text-gray-300">
            System prompt
          </summary>
          <div className="border-t border-gray-800 px-4 py-3">
            <Markdown className="text-sm text-gray-300">{systemPrompt}</Markdown>
          </div>
        </details>
      )}

      {taskStatus === "Running" && (
        <div className="flex justify-end">
          <button
            onClick={() => setAutoScroll((v) => !v)}
            className="rounded-md border border-gray-800 px-2 py-1 text-xs text-gray-500 hover:text-gray-300"
          >
            Auto-scroll {autoScroll ? "on" : "off"}
          </button>
        </div>
      )}

      <div
        ref={scrollRef}
        className="overflow-y-auto rounded-xl border border-gray-800 bg-gray-950/50 p-5"
      >
        <div className="relative space-y-4 pl-7">
          <div className="absolute bottom-0 left-[7px] top-1 w-px bg-gradient-to-b from-gray-700 via-gray-800 to-transparent" />

          {/* User prompt */}
          <article className="relative">
            <div className="absolute -left-[1.55rem] top-2.5 size-2.5 rounded-full border border-sky-400/50 bg-sky-500/30" />
            <div className="rounded-xl border border-gray-800 bg-gray-900/80 p-4">
              <p className="mb-2 text-[11px] font-medium uppercase tracking-wider text-sky-300/70">
                User
              </p>
              <Markdown className="text-sm text-gray-200">
                {taskDescription}
              </Markdown>
            </div>
          </article>

          {timeline.length === 0 ? (
            <p className="pl-1 text-sm text-gray-600">
              No runtime events yet.
            </p>
          ) : (
            timeline.map((item) => {
              if (item.kind === "assistant") {
                return (
                  <article key={item.id} className="relative">
                    <div className="absolute -left-[1.55rem] top-2.5 size-2.5 rounded-full border border-violet-400/40 bg-violet-500/25" />
                    <div className="rounded-xl border border-gray-800 bg-gray-900/80 p-4">
                      <div className="mb-2 flex items-center justify-between">
                        <p className="text-[11px] font-medium uppercase tracking-wider text-violet-300/70">
                          Assistant
                        </p>
                        <span className="text-[11px] text-gray-600">
                          {item.timestamp}
                        </span>
                      </div>
                      <Markdown className="text-[15px] leading-7 text-gray-100">
                        {item.content}
                      </Markdown>
                    </div>
                  </article>
                );
              }

              // Tool use (combined call + shell + result)
              const shellLines = item.shellOutput
                ? countLines(item.shellOutput)
                : 0;

              return (
                <article key={item.id} className="relative">
                  <div className="absolute -left-[1.55rem] top-2 size-2.5 rounded-full border border-gray-600/50 bg-gray-600/25" />

                  <div className="rounded-lg border border-gray-800/60 bg-gray-900/50 px-3 py-2">
                    <div className="flex items-center justify-between gap-3">
                      <p className="text-[13px] text-gray-300">{item.title}</p>
                      <span className="shrink-0 text-[11px] text-gray-600">
                        {item.timestamp}
                      </span>
                    </div>

                    {item.result && (
                      <p className="mt-1 text-xs text-gray-500">
                        {item.result}
                      </p>
                    )}

                    {item.shellOutput && (
                      <details className="mt-1.5">
                        <summary className="cursor-pointer text-[11px] text-gray-500 hover:text-gray-300">
                          {shellLines > 10
                            ? `Shell output (${shellLines} lines)`
                            : "Shell output"}
                        </summary>
                        <pre className="mt-1.5 max-h-60 overflow-auto whitespace-pre-wrap break-words rounded-md border border-gray-800 bg-gray-950/80 p-2.5 text-xs text-gray-400">
                          {item.shellOutput}
                        </pre>
                      </details>
                    )}
                  </div>
                </article>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
}
