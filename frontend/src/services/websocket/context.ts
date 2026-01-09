import { createContext } from "react";
import type { ConnectionState, WebSocketClient } from "./websocket-client";

export interface WebSocketContextValue {
  /**
   * Current WebSocket connection state.
   */
  connectionState: ConnectionState;
  /**
   * The WebSocket client instance for sending messages.
   */
  wsClient: WebSocketClient;
  /**
   * Force a reconnection (useful after token refresh).
   */
  reconnect: () => void;
}

export const WebSocketContext = createContext<WebSocketContextValue>(
  null as any,
);
