import { useCallback } from "react";
import { useQuery, useSpacetime, useSubscription } from "../hooks/useSpacetime";
import { Badge } from "../components/Badge";
import { tables } from "../module_bindings";

function runtimeStatusColor(tag: string): string {
  switch (tag) {
    case "Idle":
      return "bg-green-900/50 text-green-300";
    case "Busy":
      return "bg-yellow-900/50 text-yellow-300";
    case "Draining":
      return "bg-orange-900/50 text-orange-300";
    case "Offline":
      return "bg-gray-800 text-gray-400";
    default:
      return "bg-gray-800 text-gray-400";
  }
}

function StatCard({
  title,
  value,
  sub,
}: {
  title: string;
  value: number;
  sub?: string;
}) {
  return (
    <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
      <p className="text-xs font-medium uppercase tracking-wider text-gray-500">
        {title}
      </p>
      <p className="mt-1 text-2xl font-semibold text-gray-100">{value}</p>
      {sub && <p className="mt-0.5 text-xs text-gray-500">{sub}</p>}
    </div>
  );
}

const QUERY = [
  tables.agent_runtime,
  tables.agent_task,
  tables.observation,
  tables.agent_type,
];

export function DashboardPage() {
  const sub = useSubscription(
    QUERY,
    useCallback(
      (db) => [db.agent_runtime, db.agent_task, db.observation, db.agent_type],
      [],
    ),
  );
  const runtimes = useQuery(sub, (db) => Array.from(db.agent_runtime.iter()));
  const tasks = useQuery(sub, (db) => Array.from(db.agent_task.iter()));
  const observations = useQuery(sub, (db) => Array.from(db.observation.iter()));
  const agentTypes = useQuery(sub, (db) => Array.from(db.agent_type.iter()));

  if (sub.error) {
    return (
      <div className="flex h-full items-center justify-center text-red-400">
        Subscription error: {sub.error}
      </div>
    );
  }

  if (!sub.subscribed) {
    return (
      <div className="flex h-full items-center justify-center text-gray-500">
        Waiting for SpacetimeDB connection...
      </div>
    );
  }

  const onlineRuntimes = (runtimes ?? []).filter(
    (r) => r.status.tag !== "Offline",
  );
  const activeTasks = (tasks ?? []).filter(
    (t) =>
      t.status.tag !== "Completed" &&
      t.status.tag !== "Failed" &&
      t.status.tag !== "Cancelled",
  );
  const recentObs = [...(observations ?? [])]
    .sort(
      (a, b) => b.createdAt.toDate().getTime() - a.createdAt.toDate().getTime(),
    )
    .slice(0, 10);

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-semibold text-gray-100">Dashboard</h1>

      {/* Stats row */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <StatCard title="Agent Types" value={(agentTypes ?? []).length} />
        <StatCard
          title="Runtimes"
          value={onlineRuntimes.length}
          sub={`${(runtimes ?? []).length} total`}
        />
        <StatCard
          title="Active Tasks"
          value={activeTasks.length}
          sub={`${(tasks ?? []).length} total`}
        />
        <StatCard title="Observations" value={(observations ?? []).length} />
      </div>

      {/* Runtimes */}
      <section>
        <h2 className="mb-3 text-sm font-medium uppercase tracking-wider text-gray-500">
          Runtimes
        </h2>
        {(runtimes ?? []).length === 0 ? (
          <p className="text-sm text-gray-600">No runtimes registered.</p>
        ) : (
          <div className="overflow-x-auto rounded-lg border border-gray-800">
            <table className="w-full text-sm">
              <thead className="border-b border-gray-800 bg-gray-900 text-left text-xs uppercase tracking-wider text-gray-500">
                <tr>
                  <th className="px-4 py-2">Name</th>
                  <th className="px-4 py-2">Status</th>
                  <th className="px-4 py-2">Last Heartbeat</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-800/50">
                {(runtimes ?? []).map((r) => (
                  <tr key={r.identity.toHexString()}>
                    <td className="px-4 py-2 font-medium text-gray-200">
                      {r.name}
                    </td>
                    <td className="px-4 py-2">
                      <Badge
                        label={r.status.tag}
                        color={runtimeStatusColor(r.status.tag)}
                      />
                    </td>
                    <td className="px-4 py-2 text-gray-400">
                      {r.lastHeartbeat.toDate().toLocaleTimeString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      {/* Recent observations */}
      <section>
        <h2 className="mb-3 text-sm font-medium uppercase tracking-wider text-gray-500">
          Recent Observations
        </h2>
        {recentObs.length === 0 ? (
          <p className="text-sm text-gray-600">No observations yet.</p>
        ) : (
          <div className="space-y-2">
            {recentObs.map((o) => (
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
                  <span>Task #{Number(o.taskId)}</span>
                  <span>{o.createdAt.toDate().toLocaleString()}</span>
                </div>
                <p className="mt-1 text-sm text-gray-300">{o.content}</p>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
