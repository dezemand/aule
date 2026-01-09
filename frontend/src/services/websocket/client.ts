import { WebSocketClient, type ConnectionState } from "./websocket-client";
export type { ConnectionState };
import { getValidToken } from "../auth/api";
import { useAuth } from "../auth/use-auth";
import { useContext, useEffect, useState, useSyncExternalStore } from "react";
import { WebSocketContext } from "./context";

/**
 * Singleton WebSocket client instance.
 * Created once and reused across the application.
 */
export const wsClient = new WebSocketClient({
  getToken: getValidToken,
  initialRetryDelay: 1000,
  maxRetryDelay: 30000,
});

(window as any)._wsClient = wsClient; // For debugging purposes

/**
 * Connect the WebSocket client.
 * Safe to call multiple times - will only connect if disconnected.
 */
export function connectWebSocket(): void {
  wsClient.connect();
}

/**
 * Disconnect the WebSocket client and clear auth.
 */
export function disconnectWebSocket(): void {
  wsClient.disconnect();
}

/**
 * Hook to access just the WebSocket client. Throws if not connected.
 */
export function useWebSocket(): WebSocketClient {
  const { wsClient } = useContext(WebSocketContext);
  return wsClient;
}

/**
 * Hook to get the WebSocket connection state.
 */
export function useConnectionState(): ConnectionState {
  const { wsClient } = useContext(WebSocketContext);
  return useSyncExternalStore(
    (cb) => wsClient.subscribeToState(cb),
    () => wsClient.getState(),
  );
}
