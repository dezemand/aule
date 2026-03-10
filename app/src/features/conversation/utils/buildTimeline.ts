import type { RuntimeEvent } from "@/module_bindings/types";

export type TimelineItem =
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

type TurnBucket = {
  turn: number;
  toolCall: RuntimeEvent | null;
  toolResults: RuntimeEvent[];
  shellOutput: RuntimeEvent | null;
};

export function parseKeyValue(content: string): Record<string, string> {
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

export function stripToolArgLines(content: string): string {
  return content
    .replace(/^\s*\[tool_args:[^\]]+\].*$/gm, "")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}

export function extractShellCommand(action: string): string | null {
  const match = action.match(/Shell\s*\{\s*command:\s*"([\s\S]*?)"\s*\}/);
  return match?.[1] ?? null;
}

export function summarizeAction(action: string | undefined): string {
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

export function summarizeResult(result: string | undefined): string | null {
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

export function countLines(text: string): number {
  return text ? text.split("\n").length : 0;
}

export function buildTimeline(events: RuntimeEvent[]): TimelineItem[] {
  const items: TimelineItem[] = [];
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

    const mainResult = bucket.toolResults.find((r) => {
      const p = parseKeyValue(r.content);
      return !p.result?.includes("Skipped:");
    });
    const resultParsed = mainResult ? parseKeyValue(mainResult.content) : {};

    const shellContent = bucket.shellOutput?.content ?? null;
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
