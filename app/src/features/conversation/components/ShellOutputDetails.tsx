import { useState } from "react";
import { Box, Code, Collapse, UnstyledButton } from "@mantine/core";

import { countLines } from "../utils/buildTimeline";

type ShellOutputDetailsProps = {
  output: string;
};

export function ShellOutputDetails({ output }: ShellOutputDetailsProps) {
  const [opened, setOpened] = useState(false);
  const lines = countLines(output);

  return (
    <Box mt={6}>
      <UnstyledButton
        onClick={() => setOpened((v) => !v)}
        fz={11}
        c="dimmed"
        style={{ cursor: "pointer" }}
      >
        {opened ? "Hide" : "Show"} shell output
        {lines > 10 ? ` (${lines} lines)` : ""}
      </UnstyledButton>
      <Collapse in={opened}>
        <Code
          block
          mt={6}
          fz="xs"
          c="dimmed"
          style={{
            maxHeight: 240,
            overflow: "auto",
            whiteSpace: "pre-wrap",
            wordBreak: "break-word",
          }}
        >
          {output}
        </Code>
      </Collapse>
    </Box>
  );
}
