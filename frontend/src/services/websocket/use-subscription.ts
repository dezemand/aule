import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { useWebSocket } from "./client";

export interface UseSubscriptionOptions {
  queryKey: readonly unknown[];
}

function serializeKey(key: readonly unknown[]): string {
  return JSON.stringify(key);
}

const activeSubscriptions = new Map<
  string,
  UseSubscriptionOptions["queryKey"]
>();

export function useSubscription<T>(options: UseSubscriptionOptions) {
  const queryClient = useQueryClient();
  const wsClient = useWebSocket();
  const subKey = serializeKey(options.queryKey);
  const query = useQuery({
    queryKey: options.queryKey,
    queryFn: () => new Promise(() => {}), // Never resolves
    staleTime: Infinity,
  });
  const firstData = useRef(!query.isLoading);
  firstData.current = !query.isLoading;

  useEffect(() => {
    // Register subscription
    activeSubscriptions.set(subKey, options.subscribe);

    // Subscribe if already connected
    if (wsClient.getState() === "connected") {
      // wsClient.send("subscribe", options.subscribe);
    }

    // Handle incoming data
    const unsub = wsClient.addMessageHandler((envelope) => {
      if (matches(envelope, options.subscribe)) {
        queryClient.setQueryData(options.queryKey, envelope.payload);
      }
    });

    return () => {
      activeSubscriptions.delete(subKey);
      unsub();
      if (wsClient.getState() === "connected") {
        wsClient.send("unsubscribe", options.subscribe);
      }
    };
  }, [subKey]);

  return query;
}
