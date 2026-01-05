import type {
  ConnectionState,
  WebSocketClient,
} from "../websocket/websocket-client";
import { createContext } from "react";

export interface AuthContextValue {
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
  /**
   * Sign out the user.
   */
  signOut: () => void;
}

export const AuthContext = createContext<AuthContextValue | null>(null);
