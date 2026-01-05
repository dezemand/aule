import { WebSocketClient, type ConnectionState } from "./websocket-client";
import { getValidToken } from "../auth/api";
import { useAuth } from "../auth/use-auth";

/**
 * Singleton WebSocket client instance.
 * Created once and reused across the application.
 */
export const wsClient = new WebSocketClient({
  getToken: getValidToken,
  initialRetryDelay: 1000,
  maxRetryDelay: 30000,
});

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
  const { wsClient, connectionState } = useAuth();
  return wsClient;
}

/**
 * Hook to get the WebSocket connection state.
 */
export function useConnectionState(): ConnectionState {
  const { connectionState } = useAuth();
  return connectionState;
}
