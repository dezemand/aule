import { useCallback, useEffect, useRef, useSyncExternalStore } from "react";

import { subscriptionManager } from "./useSpacetimeConnection";
import type { SubscriptionDef } from "../subscription";

type SubscriptionState = {
  subscribed: boolean;
  error: string | null;
};

const NOOP_UNSUBSCRIBE = () => {};
const EMPTY_BUS = {
  subscribe: (_listener: () => void) => NOOP_UNSUBSCRIBE,
  getVersion: () => 0,
};

export function useSubscription(
  key: string,
  def: SubscriptionDef,
): SubscriptionState {
  useEffect(() => {
    subscriptionManager.ensure(key, def);
    subscriptionManager.retain(key);
    return () => subscriptionManager.release(key);
  }, [key]);

  const subscribe = useCallback(
    (listener: () => void) => {
      const bus = subscriptionManager.getBus(key);
      if (bus) return bus.subscribe(listener);
      return NOOP_UNSUBSCRIBE;
    },
    [key],
  );

  const cacheRef = useRef<{ version: number; value: SubscriptionState }>({
    version: -1,
    value: { subscribed: false, error: null },
  });

  const getSnapshot = useCallback((): SubscriptionState => {
    const bus = subscriptionManager.getBus(key);
    const version = bus?.getVersion() ?? 0;

    if (cacheRef.current.version === version) {
      return cacheRef.current.value;
    }

    const status = subscriptionManager.getStatus(key);
    const error = subscriptionManager.getError(key);
    const value = { subscribed: status === "active", error };
    cacheRef.current = { version, value };
    return value;
  }, [key]);

  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}
