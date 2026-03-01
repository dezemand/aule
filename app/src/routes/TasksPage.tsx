import { Link } from "@tanstack/react-router";
import { useCallback, useEffect, useState } from "react";
import { Badge } from "../components/Badge";
import { Markdown } from "../components/Markdown";
import { TaskAssignmentControl } from "../components/TaskAssignmentControl";
import { useQuery, useSpacetime, useSubscription } from "../hooks/useSpacetime";
import { tables } from "../module_bindings";
import { taskStatusColor } from "../utils/statusColors";

type StatusFilter = "all" | "active" | "completed" | "failed";

const QUERY = [
  tables.agent_task,
  tables.agent_type,
  tables.observation,
  tables.runtime_event,
  tables.agent_runtime,
];

function formatDuration(milliseconds: number): string {
  const totalSeconds = Math.max(0, Math.floor(milliseconds / 1000));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  if (hours > 0) {
    return `${hours}h ${minutes}m ${seconds}s`;
  }
  if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  }
  return `${seconds}s`;
}

export function TasksPage() {
  const { ctx } = useSpacetime();
  const sub = useSubscription(
    QUERY,
    useCallback(
      (db) => [
        db.agent_task,
        db.agent_type,
        db.observation,
        db.runtime_event,
        db.agent_runtime,
      ],
      [],
    ),
  );
  const tasks = useQuery(sub, (db) => Array.from(db.agent_task.iter()));
  const agentTypes = useQuery(sub, (db) => Array.from(db.agent_type.iter()));
  const observations = useQuery(sub, (db) => Array.from(db.observation.iter()));
  const runtimeEvents = useQuery(sub, (db) => Array.from(db.runtime_event.iter()));
  const runtimes = useQuery(sub, (db) => Array.from(db.agent_runtime.iter()));
  const [filter, setFilter] = useState<StatusFilter>("all");
  const [showCreate, setShowCreate] = useState(false);
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, []);

  if (!sub.subscribed) {
    return (
      <div className="flex h-full items-center justify-center text-gray-500">
        Waiting for SpacetimeDB connection...
      </div>
    );
  }

  const allTasks = tasks ?? [];
  const filtered = allTasks.filter((task) => {
    const tag = task.status.tag;
    switch (filter) {
      case "active":
        return tag === "Pending" || tag === "Assigned" || tag === "Running";
      case "completed":
        return tag === "Completed";
      case "failed":
        return tag === "Failed";
      default:
        return true;
    }
  });

  const sorted = [...filtered].sort(
    (a, b) => b.createdAt.toDate().getTime() - a.createdAt.toDate().getTime(),
  );

  const agentTypeMap = new Map(
    (agentTypes ?? []).map((agentType) => [Number(agentType.id), agentType.name]),
  );
  const runtimeNameByIdentity = new Map(
    (runtimes ?? []).map((runtime) => [runtime.identity.toHexString(), runtime.name]),
  );
  const maxTurnByTask = new Map<string, number>();
  for (const event of runtimeEvents ?? []) {
    const key = event.taskId.toString();
    maxTurnByTask.set(key, Math.max(maxTurnByTask.get(key) ?? 0, Number(event.turn)));
  }

  function getTaskObservations(taskId: bigint) {
    return (observations ?? [])
      .filter((observation) => observation.taskId === taskId)
      .sort(
        (a, b) => a.createdAt.toDate().getTime() - b.createdAt.toDate().getTime(),
      );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-gray-100">Tasks</h1>
        <button
          onClick={() => setShowCreate(!showCreate)}
          className="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-blue-500"
        >
          {showCreate ? "Cancel" : "New Task"}
        </button>
      </div>

      {showCreate && ctx && (
        <CreateTaskForm
          conn={ctx}
          agentTypes={agentTypes ?? []}
          onCreated={() => setShowCreate(false)}
        />
      )}

      <div className="flex gap-2">
        {(["all", "active", "completed", "failed"] as StatusFilter[]).map(
          (nextFilter) => (
            <button
              key={nextFilter}
              onClick={() => setFilter(nextFilter)}
              className={`rounded-md px-3 py-1 text-xs font-medium capitalize transition-colors ${
                filter === nextFilter
                  ? "bg-gray-700 text-gray-100"
                  : "text-gray-500 hover:text-gray-300"
              }`}
            >
              {nextFilter}
            </button>
          ),
        )}
      </div>

      {sorted.length === 0 ? (
        <p className="text-sm text-gray-600">No tasks found.</p>
      ) : (
        <div className="space-y-3">
          {sorted.map((task) => {
            const taskObservations = getTaskObservations(task.id);
            const latestTurn = maxTurnByTask.get(task.id.toString()) ?? 0;
            const assignedRuntimeName = task.assignedRuntime
              ? (runtimeNameByIdentity.get(task.assignedRuntime.toHexString()) ??
                "Unknown runtime")
              : "Unassigned";
            const elapsed = task.startedAt
              ? formatDuration(now - task.startedAt.toDate().getTime())
              : null;

            return (
              <div
                key={Number(task.id)}
                className="rounded-lg border border-gray-800 bg-gray-900 p-4"
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <h3 className="font-medium text-gray-200">{task.title}</h3>
                      <Badge
                        label={task.status.tag}
                        color={taskStatusColor(task.status.tag)}
                      />
                    </div>
                    <Markdown className="mt-1 text-sm text-gray-300">
                      {task.description || "(no description)"}
                    </Markdown>
                  </div>
                  <span className="text-xs text-gray-600">#{Number(task.id)}</span>
                </div>

                <div className="mt-3 flex flex-wrap gap-4 text-xs text-gray-500">
                  <span>
                    Type: {agentTypeMap.get(Number(task.agentTypeId)) ?? "Unknown"}
                  </span>
                  <span>Created: {task.createdAt.toDate().toLocaleString()}</span>
                  <span>Runtime: {assignedRuntimeName}</span>
                  {latestTurn > 0 && <span>Turn: {latestTurn}</span>}
                  {task.status.tag === "Running" && elapsed && (
                    <span className="text-yellow-300">Running for {elapsed}</span>
                  )}
                </div>

                {task.status.tag === "Running" && (
                  <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-gray-800">
                    <div
                      className="h-full rounded-full bg-yellow-400/70"
                      style={{ width: `${Math.min((latestTurn / 24) * 100, 100)}%` }}
                    />
                  </div>
                )}

                {task.result && (
                  <div className="mt-3 rounded-md border border-green-900/40 bg-green-950/20 p-3">
                    <p className="text-xs font-medium uppercase tracking-wider text-green-200/80">
                      Result
                    </p>
                    <Markdown className="mt-1 text-sm text-green-100">
                      {task.result}
                    </Markdown>
                  </div>
                )}

                <div className="mt-3">
                  <TaskAssignmentControl
                    taskId={task.id}
                    taskStatus={task.status.tag}
                    assignedRuntime={task.assignedRuntime ?? null}
                    runtimes={runtimes ?? []}
                    conn={ctx}
                    compact
                  />
                </div>

                <div className="mt-3">
                  <Link
                    to="/tasks/$taskId"
                    params={{ taskId: task.id.toString() }}
                    className="text-xs text-blue-400 hover:text-blue-300"
                  >
                    View details →
                  </Link>
                </div>

                {taskObservations.length > 0 && (
                  <div className="mt-3 space-y-1 border-t border-gray-800 pt-3">
                    <p className="text-xs font-medium uppercase tracking-wider text-gray-600">
                      Observations
                    </p>
                    {taskObservations.map((observation) => (
                      <div
                        key={Number(observation.id)}
                        className="rounded-md border border-gray-800 bg-gray-950/60 px-3 py-2"
                      >
                        <div className="flex items-start gap-2 text-xs">
                          <Badge
                            label={observation.kind.tag}
                            color={
                              observation.kind.tag === "Error"
                                ? "bg-red-900/50 text-red-300"
                                : observation.kind.tag === "Result"
                                  ? "bg-green-900/50 text-green-300"
                                  : "bg-gray-800 text-gray-300"
                            }
                          />
                          <span className="ml-auto text-gray-600">
                            {observation.createdAt.toDate().toLocaleTimeString()}
                          </span>
                        </div>
                        <Markdown className="mt-1 text-xs text-gray-300">
                          {observation.content}
                        </Markdown>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

function CreateTaskForm({
  conn,
  agentTypes,
  onCreated,
}: {
  conn: NonNullable<ReturnType<typeof useSpacetime>["ctx"]>;
  agentTypes: Array<{ id: bigint; name: string }>;
  onCreated: () => void;
}) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [agentTypeId, setAgentTypeId] = useState<string>("");

  function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    if (!title || !agentTypeId) {
      return;
    }

    conn.reducers.createTask({
      agentTypeId: BigInt(agentTypeId),
      title,
      description,
    });

    setTitle("");
    setDescription("");
    setAgentTypeId("");
    onCreated();
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="rounded-lg border border-gray-800 bg-gray-900 p-4 space-y-3"
    >
      <div>
        <label className="block text-xs font-medium text-gray-400 mb-1">
          Agent Type
        </label>
        <select
          value={agentTypeId}
          onChange={(event) => setAgentTypeId(event.currentTarget.value)}
          className="w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-1.5 text-sm text-gray-200"
          required
        >
          <option value="">Select agent type...</option>
          {agentTypes.map((agentType) => (
            <option key={Number(agentType.id)} value={agentType.id.toString()}>
              {agentType.name}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label className="block text-xs font-medium text-gray-400 mb-1">
          Title
        </label>
        <input
          type="text"
          value={title}
          onChange={(event) => setTitle(event.currentTarget.value)}
          className="w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-1.5 text-sm text-gray-200"
          required
        />
      </div>

      <div>
        <label className="block text-xs font-medium text-gray-400 mb-1">
          Description
        </label>
        <textarea
          value={description}
          onChange={(event) => setDescription(event.currentTarget.value)}
          rows={3}
          className="w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-1.5 text-sm text-gray-200"
        />
      </div>

      <button
        type="submit"
        className="rounded-md bg-blue-600 px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-blue-500"
      >
        Create Task
      </button>
    </form>
  );
}
