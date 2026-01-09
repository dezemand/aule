import {
  useQuery,
  useQueryClient,
  type QueryKey,
  type QueryState,
} from "@tanstack/react-query";
import { useEffect, useMemo, useRef, type RefObject } from "react";
import { useConnectionState, useWebSocket } from "../websocket/client";
import type { Envelope } from "../websocket/websocket-client";

function isStale(state: QueryState | undefined, staleTime: number): boolean {
  if (!state) {
    return true;
  }
  return Date.now() - state.dataUpdatedAt > staleTime;
}

export interface UseSubscriptionOptions<
  TQueryKey extends QueryKey = QueryKey,
  TQuery = any,
> {
  queryKey: TQueryKey;
  topic: string;
  query?: TQuery;
  staleTime?: number;
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

export function useSubscribe<
  TQueryKey extends QueryKey = QueryKey,
  TQuery = any,
>(
  subIdRef: RefObject<string | null>,
  {
    queryKey,
    topic,
    query,
    staleTime = 60000,
  }: UseSubscriptionOptions<TQueryKey, TQuery>,
) {
  const wsClient = useWebSocket();
  const connectionState = useConnectionState();
  const cleanup = useRef(() => {});

  const key = useMemo(
    () => ["subscription", ...queryKey],
    [topic, JSON.stringify(queryKey)],
  );
  const enabled = topic !== null && connectionState === "connected";

  const q = useQuery({
    queryKey: key,
    queryFn: async ({ client, queryKey: key }) => {
      const initial =
        !client.getQueryData(key) ||
        isStale(client.getQueryState(key), staleTime);

      cleanup.current = () => {
        wsClient.send("subscription.unsubscribe.req", {
          subscription_id: subIdRef.current,
        });
        client.setQueryData(key, null);
      };

      const res = await wsClient
        .send("subscription.subscribe.req", {
          topic,
          query,
          initial,
        })
        .response();

      subIdRef.current = (res.payload as SubscribeAckPayload)?.subscription_id;
      return res;
    },
    staleTime: 0,
    gcTime: 0,
    enabled,
  });

  useEffect(() => {
    return () => {
      cleanup.current();
    };
  }, [key]);
}

export function useMessageHandler<TQueryKey extends QueryKey = QueryKey>(
  subscriptionIdRef: RefObject<string | null>,
  queryKey: TQueryKey,
) {
  const wsClient = useWebSocket();
  const queryClient = useQueryClient();

  useEffect(() => {
    return wsClient.addMessageHandler((msg) => {
      if (
        !subscriptionIdRef.current ||
        msg.subscription_id !== subscriptionIdRef.current
      ) {
        return;
      }

      queryClient.setQueryData(queryKey, msg as any);
    });
  }, [wsClient, queryClient, JSON.stringify(queryKey)]);
}

export function useSubscription<
  TResult = any,
  TQueryKey extends QueryKey = QueryKey,
  TQuery = any,
>({
  queryKey,
  topic,
  query,
  staleTime = 60000,
}: UseSubscriptionOptions<TQueryKey, TQuery>) {
  const subscriptionIdRef = useRef(null);

  useMessageHandler(subscriptionIdRef, queryKey);
  useSubscribe(subscriptionIdRef, { queryKey, topic, query, staleTime });

  return useQuery<Envelope<TResult>>({
    queryKey,
    queryFn: () => new Promise(() => {}),
    staleTime: Infinity,
    gcTime: Infinity,
  });
}
