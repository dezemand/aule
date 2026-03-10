import { Badge, Paper, Table, Text } from "@mantine/core";

import { runtimeStatusColor } from "@/shared/utils/statusColors";

import type { AgentRuntime } from "@/module_bindings/types";

type RuntimesTableProps = {
  runtimes: AgentRuntime[];
};

export function RuntimesTable({ runtimes }: RuntimesTableProps) {
  if (runtimes.length === 0) {
    return (
      <Text size="sm" c="dimmed">
        No runtimes registered.
      </Text>
    );
  }

  return (
    <Paper withBorder radius="md" style={{ overflow: "hidden" }}>
      <Table highlightOnHover>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Name</Table.Th>
            <Table.Th>Status</Table.Th>
            <Table.Th>Last Heartbeat</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {runtimes.map((r) => (
            <Table.Tr key={r.identity.toHexString()}>
              <Table.Td fw={500}>{r.name}</Table.Td>
              <Table.Td>
                <Badge
                  size="sm"
                  variant="light"
                  color={runtimeStatusColor(r.status.tag)}
                >
                  {r.status.tag}
                </Badge>
              </Table.Td>
              <Table.Td c="dimmed">
                {r.lastHeartbeat.toDate().toLocaleTimeString()}
              </Table.Td>
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    </Paper>
  );
}
