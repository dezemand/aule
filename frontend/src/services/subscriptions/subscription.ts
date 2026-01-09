import { use, useMemo, useSyncExternalStore } from "react";
import { useConnectionState, useWebSocket } from "../websocket/client";
import type { WebSocketClient } from "../websocket/websocket-client";

interface SubscriptionResult<R> {
  data: R | null;
}

export class Subscription<Q = any, R = any> {
  private cleanHandler;

  constructor(
    private wsClient: WebSocketClient,
    readonly topic: string,
    readonly query: Q,
  ) {
    this.cleanHandler = wsClient.addMessageHandler(() => {});
  }

  subscribe(cb: () => void): () => void {
    return () => {};
  }

  getSnapshot(): SubscriptionResult<R> {
    return {
      data: null,
    };
  }

  cleanup() {
    this.cleanHandler();
  }
}

export function useSub<Q, R>(topic: string, query: Q): SubscriptionResult<R> {
  const wsClient = useWebSocket();
  const connectionState = useConnectionState();

  const sub = useMemo(
    () => new Subscription(wsClient, topic, query),
    [wsClient, topic, JSON.stringify(query)],
  );

  const result = useSyncExternalStore(
    (cb) => sub.subscribe(cb),
    () => sub.getSnapshot(),
  );

  return result;
}
