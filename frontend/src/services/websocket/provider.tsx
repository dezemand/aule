import {
  useCallback,
  useEffect,
  useMemo,
  useSyncExternalStore,
  type FC,
  type ReactNode,
} from "react";
import { WebSocketContext } from "./context";
import type { ConnectionState } from "./websocket-client";
import { connectWebSocket, disconnectWebSocket, wsClient } from "./client";
import { useAuthStore } from "../auth/store";

type WsProviderProps = {
  children: ReactNode;
  /**
   * Called when authentication fails and user should be redirected to login.
   */
  onAuthFailure?: () => void;
};

// Subscribe function for useSyncExternalStore
function subscribeToConnectionState(callback: () => void): () => void {
  return wsClient.subscribeToState(callback);
}

// Snapshot function for useSyncExternalStore
function getConnectionStateSnapshot(): ConnectionState {
  return wsClient.getState();
}

export const WsProvider: FC<WsProviderProps> = ({
  children,
  onAuthFailure,
}) => {
  // Use useSyncExternalStore to subscribe to the WebSocket client's state
  const connectionState = useSyncExternalStore(
    subscribeToConnectionState,
    getConnectionStateSnapshot,
    getConnectionStateSnapshot, // Server snapshot (same as client for now)
  );
  const auth = useAuthStore();

  // Subscribe to auth failure events
  useEffect(() => {
    if (!onAuthFailure) return;

    return wsClient.subscribeToAuthFailure(() => {
      console.log("Auth failure, redirecting to login...");
      auth.setToken(null);
      onAuthFailure();
    });
  }, [onAuthFailure, auth]);

  // Connect on mount (safe to call multiple times)
  useEffect(() => {
    connectWebSocket();
  }, []);

  // Reconnect when window regains focus if disconnected
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (
        document.visibilityState === "visible" &&
        wsClient.getState() === "disconnected"
      ) {
        connectWebSocket();
      }
    };

    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, []);

  const reconnect = useCallback(() => {
    disconnectWebSocket();
    connectWebSocket();
  }, []);

  const value = useMemo(
    () => ({
      wsClient,
      connectionState,
      reconnect,
    }),
    [wsClient, connectionState, reconnect],
  );

  return (
    <WebSocketContext.Provider value={value}>
      {children}
    </WebSocketContext.Provider>
  );
};
