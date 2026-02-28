import { useState } from "react";
import { useQuery, useSpacetime } from "../hooks/useSpacetime";
import { Badge } from "../components/Badge";

function taskStatusColor(tag: string): string {
  switch (tag) {
    case "Pending":
      return "bg-gray-800 text-gray-300";
    case "Assigned":
      return "bg-blue-900/50 text-blue-300";
    case "Running":
      return "bg-yellow-900/50 text-yellow-300";
    case "Completed":
      return "bg-green-900/50 text-green-300";
    case "Failed":
      return "bg-red-900/50 text-red-300";
    case "Cancelled":
      return "bg-gray-800 text-gray-500";
    default:
      return "bg-gray-800 text-gray-400";
  }
}

type StatusFilter = "all" | "active" | "completed" | "failed";

export function TasksPage() {
  const { conn, subscribed } = useSpacetime();
  const [filter, setFilter] = useState<StatusFilter>("all");
  const [showCreate, setShowCreate] = useState(false);

  const tasks = useQuery((db) => Array.from(db.agent_task.iter()));
  const agentTypes = useQuery((db) => Array.from(db.agent_type.iter()));
  const observations = useQuery((db) => Array.from(db.observation.iter()));

  if (!subscribed) {
    return (
      <div className="flex h-full items-center justify-center text-gray-500">
        Waiting for SpacetimeDB connection...
      </div>
    );
  }

  const allTasks = tasks ?? [];
  const filtered = allTasks.filter((t) => {
    const tag = t.status.tag;
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
    (a, b) =>
      b.createdAt.toDate().getTime() - a.createdAt.toDate().getTime()
  );

  const agentTypeMap = new Map(
    (agentTypes ?? []).map((at) => [Number(at.id), at.name])
  );

  function getTaskObservations(taskId: bigint) {
    return (observations ?? [])
      .filter((o) => o.taskId === taskId)
      .sort(
        (a, b) =>
          a.createdAt.toDate().getTime() - b.createdAt.toDate().getTime()
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

      {/* Create form */}
      {showCreate && conn && (
        <CreateTaskForm
          conn={conn}
          agentTypes={agentTypes ?? []}
          onCreated={() => setShowCreate(false)}
        />
      )}

      {/* Filters */}
      <div className="flex gap-2">
        {(["all", "active", "completed", "failed"] as StatusFilter[]).map(
          (f) => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={`rounded-md px-3 py-1 text-xs font-medium capitalize transition-colors ${
                filter === f
                  ? "bg-gray-700 text-gray-100"
                  : "text-gray-500 hover:text-gray-300"
              }`}
            >
              {f}
            </button>
          )
        )}
      </div>

      {/* Task list */}
      {sorted.length === 0 ? (
        <p className="text-sm text-gray-600">No tasks found.</p>
      ) : (
        <div className="space-y-3">
          {sorted.map((t) => {
            const obs = getTaskObservations(t.id);
            return (
              <div
                key={Number(t.id)}
                className="rounded-lg border border-gray-800 bg-gray-900 p-4"
              >
                <div className="flex items-start justify-between">
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="font-medium text-gray-200">{t.title}</h3>
                      <Badge
                        label={t.status.tag}
                        color={taskStatusColor(t.status.tag)}
                      />
                    </div>
                    <p className="mt-1 text-sm text-gray-400">
                      {t.description}
                    </p>
                  </div>
                  <span className="text-xs text-gray-600">
                    #{Number(t.id)}
                  </span>
                </div>
                <div className="mt-2 flex gap-4 text-xs text-gray-500">
                  <span>
                    Type:{" "}
                    {agentTypeMap.get(Number(t.agentTypeId)) ?? "Unknown"}
                  </span>
                  <span>
                    Created: {t.createdAt.toDate().toLocaleString()}
                  </span>
                  {t.result && (
                    <span className="text-green-400">
                      Result: {t.result}
                    </span>
                  )}
                </div>

                {/* Observations */}
                {obs.length > 0 && (
                  <div className="mt-3 space-y-1 border-t border-gray-800 pt-3">
                    <p className="text-xs font-medium uppercase tracking-wider text-gray-600">
                      Observations
                    </p>
                    {obs.map((o) => (
                      <div
                        key={Number(o.id)}
                        className="flex items-start gap-2 text-xs"
                      >
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
                        <span className="text-gray-400">{o.content}</span>
                        <span className="ml-auto text-gray-600">
                          {o.createdAt.toDate().toLocaleTimeString()}
                        </span>
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

// -- Create Task Form --

function CreateTaskForm({
  conn,
  agentTypes,
  onCreated,
}: {
  conn: NonNullable<ReturnType<typeof useSpacetime>["conn"]>;
  agentTypes: Array<{ id: bigint; name: string }>;
  onCreated: () => void;
}) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [agentTypeId, setAgentTypeId] = useState<string>("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!title || !agentTypeId) return;

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
          onChange={(e) => setAgentTypeId(e.currentTarget.value)}
          className="w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-1.5 text-sm text-gray-200"
          required
        >
          <option value="">Select agent type...</option>
          {agentTypes.map((at) => (
            <option key={Number(at.id)} value={at.id.toString()}>
              {at.name}
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
          onChange={(e) => setTitle(e.currentTarget.value)}
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
          onChange={(e) => setDescription(e.currentTarget.value)}
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
