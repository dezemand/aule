import { Group } from "@mantine/core";
import { ConnectionBadge } from "@/shared/components/ConnectionBadge/ConnectionBadge";
import classes from "./TopBar.module.css";

export function TopBar() {
  return (
    <Group className={classes.topBar} justify="flex-end">
      <ConnectionBadge />
    </Group>
  );
}
