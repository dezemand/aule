import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useRef } from "react";
import { useConnectionState, useWebSocket } from "../websocket/client";

export interface UseSubscriptionOptions {
  topic: string;
  query?: any;
}

export interface UseSubscriptionResult<T> {
  data: T | undefined;
  isLoading: boolean;
  isSubscribed: boolean;
  subscriptionId: string | null;
}

interface SubscribeAckPayload {
  subscription_id: string;
}

function subscriptionQueryKey(topic: string, query: any): readonly unknown[] {
  return ["subscription", "request", topic, query] as const;
}

function contentQueryKey(topic: string, query: any): readonly unknown[] {
  return ["subscription", "content", topic, query] as const;
}

export function useSubscribe<Q = any>(topic: string, query: Q) {
  const wsClient = useWebSocket();
  const connectionState = useConnectionState();
  const queryClient = useQueryClient();

  const key = useMemo(
    () => subscriptionQueryKey(topic, query),
    [topic, JSON.stringify(query)],
  );
  const enabled = topic !== null && connectionState === "connected";

  const subscriptionQuery = useQuery({
    queryKey: key,
    queryFn: ({ client }) => {
      const initial = true;
      // Check if data is present and not stale

      return wsClient
        .send("subscription.subscribe.req", {
          topic: topic,
          query: query,
          initial,
        })
        .response();
    },
    staleTime: 0,
    gcTime: 0,
    enabled,
  });

  const subscriptionId = useMemo(() => {
    if (!subscriptionQuery.data?.payload) return null;
    const payload = subscriptionQuery.data.payload as SubscribeAckPayload;
    return payload.subscription_id ?? null;
  }, [subscriptionQuery.data]);

  useEffect(() => {
    return () => {
      if (subscriptionId && wsClient.getState() === "connected") {
        wsClient.send("subscription.unsubscribe.req", {
          subscription_id: subscriptionId,
        });
        queryClient.setQueryData(key, null);
      }
    };
  }, [subscriptionId, key]);

  return subscriptionId;
}

export function useSubscription<T>(options: UseSubscriptionOptions) {
  const wsClient = useWebSocket();
  const queryClient = useQueryClient();
  const subscriptionId = useSubscribe(options.topic, options.query);
  const subIdRef = useRef<string | null>(subscriptionId);
  subIdRef.current = subscriptionId;

  const key = useMemo(
    () => contentQueryKey(options.topic, options.query),
    [options.topic, JSON.stringify(options.query)],
  );

  useEffect(() => {
    return wsClient.addMessageHandler((msg) => {
      if (msg.subscription_id === subIdRef.current) {
        console.log("Received message for subscription:", msg);
        queryClient.setQueryData(key, msg);
      }
    });
  }, [wsClient, queryClient, key]);

  return useQuery({
    queryKey: key,
    queryFn: () => new Promise(() => {}),
    enabled: subscriptionId !== null,
  });
}
