import type { SubscriptionHandle } from "@/module_bindings";

export type { SubscriptionHandle as StSubscription };

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type TableRef = any;

export type SubscriptionDef = {
  query: TableRef[];
  tables: string[];
};
