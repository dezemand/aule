import { Paper, Text } from "@mantine/core";

type StatCardProps = {
  title: string;
  value: number;
  sub?: string;
};

export function StatCard({ title, value, sub }: StatCardProps) {
  return (
    <Paper withBorder radius="md" p="md">
      <Text size="xs" fw={500} tt="uppercase" lts={0.5} c="dimmed">
        {title}
      </Text>
      <Text size="xl" fw={600} mt={4}>
        {value}
      </Text>
      {sub && (
        <Text size="xs" c="dimmed" mt={2}>
          {sub}
        </Text>
      )}
    </Paper>
  );
}
