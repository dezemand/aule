import { Link, useParams } from "@tanstack/react-router";
import { useCallback, useState } from "react";
import { Badge } from "../components/Badge";
import { useQuery, useSubscription } from "../hooks/useSpacetime";
import { tables } from "../module_bindings";
import { taskStatusColor } from "../utils/statusColors";

function runtimeEventColor(tag: string): string {
  switch (tag) {
    case "LlmResponse":
      return "bg-indigo-900/50 text-indigo-300";
    case "ToolCall":
      return "bg-cyan-900/50 text-cyan-300";
    case "ToolResult":
      return "bg-emerald-900/50 text-emerald-300";
    case "ShellOutput":
      return "bg-amber-900/50 text-amber-300";
    default:
      return "bg-gray-800 text-gray-300";
  }
}

export function TaskDetailsPage() {
  const { taskId } = useParams({ from: "/tasks/$taskId" });
  const [logsOpen, setLogsOpen] = useState(true);

  const sub = useSubscription(
    [
      tables.agent_task,
      tables.agent_type,
      tables.observation,
      tables.runtime_event,
    ],
    useCallback(
      (db) => [db.agent_task, db.agent_type, db.observation, db.runtime_event],
      [],
    ),
  );

  const tasks = useQuery(sub, (db) => Array.from(db.agent_task.iter()));
  const agentTypes = useQuery(sub, (db) => Array.from(db.agent_type.iter()));
  const observations = useQuery(sub, (db) => Array.from(db.observation.iter()));
  const runtimeEvents = useQuery(sub, (db) => Array.from(db.runtime_event.iter()));

  if (!sub.subscribed) {
    return (
      <div className="flex h-full items-center justify-center text-gray-500">
        Waiting for SpacetimeDB connection...
      </div>
    );
  }

  let parsedTaskId: bigint;
  try {
    parsedTaskId = BigInt(taskId);
  } catch {
    return (
      <div className="space-y-4">
        <Link to="/tasks" className="text-sm text-blue-400 hover:text-blue-300">
          ← Back to tasks
        </Link>
        <p className="text-sm text-red-400">Invalid task id: {taskId}</p>
      </div>
    );
  }

  const task = (tasks ?? []).find((t) => t.id === parsedTaskId);
  const agentTypeMap = new Map(
    (agentTypes ?? []).map((at) => [Number(at.id), at.name]),
  );

  if (!task) {
    return (
      <div className="space-y-4">
        <Link to="/tasks" className="text-sm text-blue-400 hover:text-blue-300">
          ← Back to tasks
        </Link>
        <p className="text-sm text-gray-500">Task #{taskId} was not found.</p>
      </div>
    );
  }

  const taskObservations = (observations ?? [])
    .filter((o) => o.taskId === task.id)
    .sort((a, b) => a.createdAt.toDate().getTime() - b.createdAt.toDate().getTime());

  const taskRuntimeEvents = (runtimeEvents ?? [])
    .filter((e) => e.taskId === task.id)
    .sort((a, b) => a.createdAt.toDate().getTime() - b.createdAt.toDate().getTime());

  return (
    <div className="space-y-6">
      <div className="space-y-3">
        <Link to="/tasks" className="text-sm text-blue-400 hover:text-blue-300">
          ← Back to tasks
        </Link>

        <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-xl font-semibold text-gray-100">{task.title}</h1>
                <Badge label={task.status.tag} color={taskStatusColor(task.status.tag)} />
              </div>
              <p className="mt-2 text-sm text-gray-400">{task.description}</p>
            </div>
            <span className="text-xs text-gray-600">#{Number(task.id)}</span>
          </div>

          <div className="mt-4 grid gap-2 text-xs text-gray-500 sm:grid-cols-2">
            <div>
              Agent type: {agentTypeMap.get(Number(task.agentTypeId)) ?? "Unknown"}
            </div>
            <div>Created: {task.createdAt.toDate().toLocaleString()}</div>
            <div>
              Started: {task.startedAt ? task.startedAt.toDate().toLocaleString() : "-"}
            </div>
            <div>
              Completed: {task.completedAt ? task.completedAt.toDate().toLocaleString() : "-"}
            </div>
          </div>

          {task.result && (
            <div className="mt-4 rounded-md border border-gray-800 bg-gray-950/70 p-3">
              <p className="text-xs font-medium uppercase tracking-wider text-gray-500">
                Result
              </p>
              <p className="mt-1 text-sm text-gray-300 whitespace-pre-wrap">{task.result}</p>
            </div>
          )}
        </div>
      </div>

      <section className="space-y-3">
        <h2 className="text-sm font-medium uppercase tracking-wider text-gray-500">
          Observations
        </h2>

        {taskObservations.length === 0 ? (
          <p className="text-sm text-gray-600">No observations yet.</p>
        ) : (
          <div className="space-y-2">
            {taskObservations.map((o) => (
              <div
                key={Number(o.id)}
                className="rounded-lg border border-gray-800 bg-gray-900 px-4 py-3"
              >
                <div className="flex items-center gap-2 text-xs text-gray-500">
                  <Badge
                    label={o.kind.tag}
                    color={
                      o.kind.tag === "Error"
                        ? "bg-red-900/50 text-red-300"
                        : o.kind.tag === "Result"
                          ? "bg-green-900/50 text-green-300"
                          : "bg-gray-800 text-gray-300"
                    }
                  />
                  <span>{o.createdAt.toDate().toLocaleString()}</span>
                </div>
                <p className="mt-1 text-sm text-gray-300 whitespace-pre-wrap">{o.content}</p>
              </div>
            ))}
          </div>
        )}
      </section>

      <section className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-medium uppercase tracking-wider text-gray-500">
            Logs
          </h2>
          <button
            onClick={() => setLogsOpen((open) => !open)}
            className="rounded-md border border-gray-700 px-2.5 py-1 text-xs text-gray-300 hover:bg-gray-800"
          >
            {logsOpen ? "Hide" : "Show"} ({taskRuntimeEvents.length})
          </button>
        </div>

        {logsOpen &&
          (taskRuntimeEvents.length === 0 ? (
            <p className="text-sm text-gray-600">No runtime logs yet.</p>
          ) : (
            <div className="space-y-3">
              {taskRuntimeEvents.map((event) => (
                <div
                  key={event.id}
                  className="rounded-lg border border-gray-800 bg-gray-950/80 p-3"
                >
                  <div className="flex flex-wrap items-center gap-2 text-xs text-gray-500">
                    <Badge
                      label={event.eventType.tag}
                      color={runtimeEventColor(event.eventType.tag)}
                    />
                    <span>Turn {event.turn}</span>
                    <span>{event.updatedAt.toDate().toLocaleTimeString()}</span>
                  </div>
                  <pre className="mt-2 max-h-64 overflow-auto whitespace-pre-wrap break-words text-xs text-gray-300">
                    {event.content || "(empty)"}
                  </pre>
                </div>
              ))}
            </div>
          ))}
      </section>
    </div>
  );
}
