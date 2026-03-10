import { useCallback, useRef, useSyncExternalStore } from "react";

import { subscriptionManager } from "./useSpacetimeConnection";
import type { DbConnection } from "@/module_bindings";

type DbHandle = DbConnection["db"];

const NOOP_UNSUBSCRIBE = () => {};
const EMPTY_BUS = {
  subscribe: (_listener: () => void) => NOOP_UNSUBSCRIBE,
  getVersion: () => 0,
};

export function useQuery<T>(
  key: string,
  selector: (db: DbHandle) => T,
): T | undefined {
  const bus = subscriptionManager.getBus(key) ?? EMPTY_BUS;
  const cacheRef = useRef<{ version: number; value: T | undefined }>({
    version: -1,
    value: undefined,
  });

  const getSnapshot = useCallback(() => {
    const version = bus.getVersion();
    if (cacheRef.current.version === version) {
      return cacheRef.current.value;
    }

    const conn = subscriptionManager.getConnection();
    if (!conn) {
      cacheRef.current = { version, value: undefined };
      return undefined;
    }

    const value = selector(conn.db);
    cacheRef.current = { version, value };
    return value;
  }, [bus, selector]);

  return useSyncExternalStore(bus.subscribe, getSnapshot, () => undefined);
}
