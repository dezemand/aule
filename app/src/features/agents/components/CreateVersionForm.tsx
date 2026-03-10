import { useState } from "react";
import { Button, Paper, Stack, TextInput, Textarea } from "@mantine/core";

import type { DbConnection } from "@/module_bindings";

type CreateVersionFormProps = {
  conn: DbConnection;
  agentTypeId: bigint;
  onCreated: () => void;
};

export function CreateVersionForm({ conn, agentTypeId, onCreated }: CreateVersionFormProps) {
  const [version, setVersion] = useState("");
  const [systemPrompt, setSystemPrompt] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!version || !systemPrompt) return;

    conn.reducers.createAgentTypeVersion({
      agentTypeId,
      version,
      systemPrompt,
    });
    setVersion("");
    setSystemPrompt("");
    onCreated();
  }

  return (
    <Paper
      component="form"
      onSubmit={handleSubmit}
      radius="sm"
      p="sm"
      bg="dark.6"
      mb="sm"
    >
      <Stack gap="xs">
        <TextInput
          label="Version"
          size="xs"
          placeholder="e.g. 0.1.0"
          value={version}
          onChange={(e) => setVersion(e.currentTarget.value)}
          required
        />
        <Textarea
          label="System Prompt"
          size="xs"
          rows={4}
          value={systemPrompt}
          onChange={(e) => setSystemPrompt(e.currentTarget.value)}
          required
        />
        <Button type="submit" size="xs">
          Add Version
        </Button>
      </Stack>
    </Paper>
  );
}
