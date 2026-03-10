import type { DbConnection, ErrorContext } from "@/module_bindings";
import { createPromise } from "@/shared/utils/promise";
import { SubscriptionManager } from "../subscriptionManager";
import { create } from "zustand";

export const subscriptionManager = new SubscriptionManager({ graceTTL: 30_000 });

type State = {
  connectionPromise: Promise<DbConnection>;

  setConnection(conn: DbConnection): void;
  setConnectionError(error: ErrorContext): void;
  disconnect(): void;
};

export const useSpacetimeConnection = create<State>((set) => {
  let { promise, resolve, reject } = createPromise<DbConnection>();

  return {
    connectionPromise: promise,

    setConnection(conn: DbConnection) {
      resolve!(conn);
      subscriptionManager.setConnection(conn);
    },

    setConnectionError(error: ErrorContext) {
      reject!(error);
    },

    disconnect() {
      subscriptionManager.clearConnection();

      const newPromise = createPromise<DbConnection>();
      resolve = newPromise.resolve;
      reject = newPromise.reject;
      set({ connectionPromise: newPromise.promise });
    },
  };
});
