import { useState } from "react";
import { Button, Paper, Stack, TextInput, Textarea } from "@mantine/core";

import type { DbConnection } from "@/module_bindings";

type CreateAgentTypeFormProps = {
  conn: DbConnection;
  onCreated: () => void;
};

export function CreateAgentTypeForm({ conn, onCreated }: CreateAgentTypeFormProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name) return;

    conn.reducers.createAgentType({ name, description });
    setName("");
    setDescription("");
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
        <TextInput
          label="Name"
          size="sm"
          value={name}
          onChange={(e) => setName(e.currentTarget.value)}
          required
        />
        <Textarea
          label="Description"
          size="sm"
          rows={2}
          value={description}
          onChange={(e) => setDescription(e.currentTarget.value)}
        />
        <Button type="submit" size="sm">
          Create
        </Button>
      </Stack>
    </Paper>
  );
}
