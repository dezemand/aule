import { useEffect, useMemo, useRef, useState } from "react";
import {
  Box,
  Collapse,
  Group,
  Paper,
  Stack,
  Text,
  UnstyledButton,
} from "@mantine/core";

import { Markdown } from "@/shared/components/Markdown/Markdown";

import { TimelineEntry } from "./components/TimelineEntry";
import { buildTimeline } from "./utils/buildTimeline";

import type { RuntimeEvent } from "@/module_bindings/types";

type ConversationViewProps = {
  events: RuntimeEvent[];
  taskDescription: string;
  taskStatus: string;
  systemPrompt?: string;
};

export function ConversationView({
  events,
  taskDescription,
  taskStatus,
  systemPrompt,
}: ConversationViewProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);
  const [systemPromptOpened, setSystemPromptOpened] = useState(false);

  const timeline = useMemo(() => buildTimeline(events), [events]);

  const lastEventUpdate =
    events.length > 0
      ? (events[events.length - 1]?.updatedAt.toDate().getTime() ?? 0)
      : 0;

  useEffect(() => {
    if (taskStatus !== "Running" || !autoScroll || !scrollRef.current) return;
    scrollRef.current.scrollTo({
      top: scrollRef.current.scrollHeight,
      behavior: "smooth",
    });
  }, [taskStatus, autoScroll, events.length, lastEventUpdate]);

  return (
    <Stack gap="sm">
      {systemPrompt && (
        <Paper withBorder radius="md" style={{ overflow: "hidden" }}>
          <UnstyledButton
            onClick={() => setSystemPromptOpened((v) => !v)}
            px="md"
            py="xs"
            w="100%"
          >
            <Text size="xs" c="dimmed">
              {systemPromptOpened ? "Hide" : "Show"} system prompt
            </Text>
          </UnstyledButton>
          <Collapse in={systemPromptOpened}>
            <Box
              px="md"
              py="sm"
              style={{ borderTop: "1px solid var(--mantine-color-dark-4)" }}
            >
              <Markdown>{systemPrompt}</Markdown>
            </Box>
          </Collapse>
        </Paper>
      )}

      {taskStatus === "Running" && (
        <Group justify="flex-end">
          <UnstyledButton
            onClick={() => setAutoScroll((v) => !v)}
            px="xs"
            py={4}
            fz="xs"
            c="dimmed"
            style={{
              border: "1px solid var(--mantine-color-dark-4)",
              borderRadius: "var(--mantine-radius-sm)",
            }}
          >
            Auto-scroll {autoScroll ? "on" : "off"}
          </UnstyledButton>
        </Group>
      )}

      <Paper
        ref={scrollRef}
        withBorder
        radius="lg"
        p="lg"
        style={{ overflowY: "auto" }}
      >
        <Box pl={28} pos="relative">
          <Box
            pos="absolute"
            left={7}
            top={4}
            bottom={0}
            w={1}
            style={{
              background:
                "linear-gradient(to bottom, var(--mantine-color-dark-3), var(--mantine-color-dark-5), transparent)",
            }}
          />

          <Stack gap="md">
            <Box pos="relative">
              <Box
                pos="absolute"
                left={-25}
                top={10}
                w={10}
                h={10}
                style={{
                  borderRadius: "50%",
                  border: "1px solid var(--mantine-color-blue-7)",
                  background: "rgba(56, 139, 253, 0.2)",
                }}
              />
              <Paper withBorder radius="lg" p="md">
                <Text
                  fz={11}
                  fw={500}
                  tt="uppercase"
                  lts={0.5}
                  c="blue.4"
                  mb="xs"
                >
                  User
                </Text>
                <Markdown>{taskDescription}</Markdown>
              </Paper>
            </Box>

            {timeline.length === 0 ? (
              <Text size="sm" c="dimmed" pl={4}>
                No runtime events yet.
              </Text>
            ) : (
              timeline.map((item) => (
                <TimelineEntry key={item.id} item={item} />
              ))
            )}
          </Stack>
        </Box>
      </Paper>
    </Stack>
  );
}
