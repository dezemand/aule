import type { MantineColor } from "@mantine/core";

export function taskStatusColor(tag: string): MantineColor {
  switch (tag) {
    case "Pending":
      return "gray";
    case "Assigned":
      return "blue";
    case "Running":
      return "yellow";
    case "Completed":
      return "green";
    case "Failed":
      return "red";
    case "Cancelled":
      return "gray";
    default:
      return "gray";
  }
}

export function runtimeStatusColor(tag: string): MantineColor {
  switch (tag) {
    case "Idle":
      return "green";
    case "Busy":
      return "yellow";
    case "Draining":
      return "orange";
    case "Offline":
      return "gray";
    default:
      return "gray";
  }
}

export function observationKindColor(tag: string): MantineColor {
  switch (tag) {
    case "Error":
      return "red";
    case "Result":
      return "green";
    default:
      return "gray";
  }
}

export function versionStatusColor(tag: string): MantineColor {
  switch (tag) {
    case "Draft":
      return "gray";
    case "Testing":
      return "yellow";
    case "Active":
      return "green";
    case "Deprecated":
      return "orange";
    case "Retired":
      return "gray";
    default:
      return "gray";
  }
}
