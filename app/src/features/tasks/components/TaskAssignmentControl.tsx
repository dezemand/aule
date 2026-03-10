import { useEffect, useMemo, useState } from "react";
import { Button, Group, NativeSelect, Stack, Text } from "@mantine/core";
import type { Identity } from "spacetimedb";

import type { DbConnection } from "@/module_bindings";

type RuntimeOption = {
  identity: Identity;
  name: string;
  status: { tag: string };
};

type TaskAssignmentControlProps = {
  taskId: bigint;
  taskStatus: string;
  assignedRuntime: Identity | null;
  runtimes: RuntimeOption[];
  conn: DbConnection | null;
  compact?: boolean;
};

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
    () =>
      new Map(
        runtimes.map((runtime) => [runtime.identity.toHexString(), runtime]),
      ),
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
    ? (runtimeByHex.get(assignedRuntime.toHexString())?.name ??
      "Unknown runtime")
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
        assignError instanceof Error
          ? assignError.message
          : "Failed to assign task",
      );
    }
  }

  const canAssign = taskStatus === "Pending" && Boolean(conn);

  return (
    <Stack gap={compact ? 6 : 8}>
      <Text size="xs" c="dimmed">
        Assigned runtime: {assignedRuntimeName ?? "Unassigned"}
      </Text>

      {canAssign && (
        <Stack gap={compact ? 6 : 8}>
          {idleRuntimes.length === 0 ? (
            <Text size="xs" c="yellow.3">
              No idle runtimes available
            </Text>
          ) : (
            <Group gap="xs" wrap="wrap">
              <NativeSelect
                size="xs"
                value={selectedRuntimeHex}
                onChange={(event) =>
                  setSelectedRuntimeHex(event.currentTarget.value)
                }
                data={idleRuntimes.map((runtime) => ({
                  value: runtime.identity.toHexString(),
                  label: runtime.name,
                }))}
              />

              <Button
                size="xs"
                variant="light"
                onClick={() =>
                  assign(
                    selectedRuntimeHex
                      ? runtimeByHex.get(selectedRuntimeHex)?.identity
                      : undefined,
                  )
                }
              >
                Assign
              </Button>

              <Button
                size="xs"
                variant="default"
                onClick={() => assign(idleRuntimes[0]?.identity)}
              >
                Auto-assign
              </Button>
            </Group>
          )}

          {error && (
            <Text size="xs" c="red.3">
              {error}
            </Text>
          )}
        </Stack>
      )}
    </Stack>
  );
}
