import { useState } from "react";
import { Box, Code, Collapse, UnstyledButton } from "@mantine/core";

import { countLines } from "../utils/buildTimeline";

type ShellOutputDetailsProps = {
  output: string;
};

export function ShellOutputDetails({ output }: ShellOutputDetailsProps) {
  const lines = countLines(output);
  const collapsible = lines > 10;
  const [opened, setOpened] = useState(!collapsible);

  return (
    <Box mt={6}>
      {collapsible && (
        <UnstyledButton
          onClick={() => setOpened((v) => !v)}
          fz={11}
          c="dimmed"
          style={{ cursor: "pointer" }}
        >
          {opened ? "Hide" : "Show"} shell output ({lines} lines)
        </UnstyledButton>
      )}
      <Collapse in={collapsible ? opened : true}>
        <Code
          block
          mt={6}
          fz="xs"
          c="dimmed"
          style={{
            maxHeight: 240,
            overflow: "auto",
            whiteSpace: "pre",
          }}
        >
          {output}
        </Code>
      </Collapse>
    </Box>
  );
}
