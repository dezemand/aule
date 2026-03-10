import { useState } from "react";
import {
  Button,
  NativeSelect,
  Paper,
  Stack,
  TextInput,
  Textarea,
} from "@mantine/core";

import type { DbConnection } from "@/module_bindings";

type CreateTaskFormProps = {
  conn: DbConnection;
  agentTypes: Array<{ id: bigint; name: string }>;
  onCreated: () => void;
};

export function CreateTaskForm({
  conn,
  agentTypes,
  onCreated,
}: CreateTaskFormProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [agentTypeId, setAgentTypeId] = useState<string>("");

  function handleSubmit(event: React.SubmitEvent<HTMLFormElement>) {
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
    <Paper
      component="form"
      onSubmit={handleSubmit}
      withBorder
      radius="md"
      p="md"
    >
      <Stack gap="sm">
        <NativeSelect
          label="Agent Type"
          size="sm"
          value={agentTypeId}
          onChange={(event) => setAgentTypeId(event.currentTarget.value)}
          data={[
            { value: "", label: "Select agent type..." },
            ...agentTypes.map((agentType) => ({
              value: agentType.id.toString(),
              label: agentType.name,
            })),
          ]}
          required
        />
        <TextInput
          label="Title"
          size="sm"
          value={title}
          onChange={(event) => setTitle(event.currentTarget.value)}
          required
        />
        <Textarea
          label="Description"
          size="sm"
          rows={3}
          value={description}
          onChange={(event) => setDescription(event.currentTarget.value)}
        />
        <Button type="submit" size="sm">
          Create Task
        </Button>
      </Stack>
    </Paper>
  );
}
