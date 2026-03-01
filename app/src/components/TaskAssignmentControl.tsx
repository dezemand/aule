import type { Identity } from "spacetimedb";
import { useEffect, useMemo, useState } from "react";
import type { DbConnection } from "../module_bindings";

interface RuntimeOption {
  identity: Identity;
  name: string;
  status: { tag: string };
}

interface TaskAssignmentControlProps {
  taskId: bigint;
  taskStatus: string;
  assignedRuntime: Identity | null;
  runtimes: RuntimeOption[];
  conn: DbConnection | null;
  compact?: boolean;
}

export function TaskAssignmentControl({
  taskId,
  taskStatus,
  assignedRuntime,
  runtimes,
  conn,
  compact = false,
}: TaskAssignmentControlProps) {
  const idleRuntimes = useMemo(
    () => runtimes.filter((runtime) => runtime.status.tag === "Idle"),
    [runtimes],
  );
  const runtimeByHex = useMemo(
    () => new Map(runtimes.map((runtime) => [runtime.identity.toHexString(), runtime])),
    [runtimes],
  );

  const [selectedRuntimeHex, setSelectedRuntimeHex] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const current = selectedRuntimeHex
      ? runtimeByHex.get(selectedRuntimeHex)
      : null;
    if (current?.status.tag === "Idle") {
      return;
    }
    setSelectedRuntimeHex(idleRuntimes[0]?.identity.toHexString() ?? "");
  }, [idleRuntimes, runtimeByHex, selectedRuntimeHex]);

  const assignedRuntimeName = assignedRuntime
    ? (runtimeByHex.get(assignedRuntime.toHexString())?.name ?? "Unknown runtime")
    : null;

  function assign(runtimeIdentity: Identity | undefined) {
    if (!conn || !runtimeIdentity) {
      return;
    }
    try {
      conn.reducers.assignTask({
        taskId,
        runtimeIdentity,
      });
      setError(null);
    } catch (assignError) {
      setError(
        assignError instanceof Error ? assignError.message : "Failed to assign task",
      );
    }
  }

  const canAssign = taskStatus === "Pending" && Boolean(conn);

  return (
    <div className={compact ? "space-y-1.5" : "space-y-2"}>
      <p className="text-xs text-gray-500">
        Assigned runtime: {assignedRuntimeName ?? "Unassigned"}
      </p>

      {canAssign && (
        <div className={compact ? "space-y-1.5" : "space-y-2"}>
          {idleRuntimes.length === 0 ? (
            <p className="text-xs text-amber-300">No idle runtimes available</p>
          ) : (
            <div className="flex flex-wrap items-center gap-2">
              <select
                value={selectedRuntimeHex}
                onChange={(event) => setSelectedRuntimeHex(event.currentTarget.value)}
                className="rounded-md border border-gray-700 bg-gray-800 px-2 py-1 text-xs text-gray-200"
              >
                {idleRuntimes.map((runtime) => (
                  <option
                    key={runtime.identity.toHexString()}
                    value={runtime.identity.toHexString()}
                  >
                    {runtime.name}
                  </option>
                ))}
              </select>

              <button
                onClick={() =>
                  assign(
                    selectedRuntimeHex
                      ? runtimeByHex.get(selectedRuntimeHex)?.identity
                      : undefined,
                  )
                }
                className="rounded-md border border-blue-500/40 bg-blue-500/10 px-2.5 py-1 text-xs font-medium text-blue-200 hover:bg-blue-500/20"
              >
                Assign
              </button>

              <button
                onClick={() => assign(idleRuntimes[0]?.identity)}
                className="rounded-md border border-gray-700 px-2.5 py-1 text-xs font-medium text-gray-200 hover:bg-gray-800"
              >
                Auto-assign
              </button>
            </div>
          )}

          {error && <p className="text-xs text-red-300">{error}</p>}
        </div>
      )}
    </div>
  );
}
