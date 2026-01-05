import {
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useSyncExternalStore,
  type FC,
  type ReactNode,
} from "react";
import { type ConnectionState, type WebSocketClient } from "@/lib/websocket";
import { wsClient, connectWebSocket, disconnectWebSocket } from "./websocket";
import { AuthContext, type AuthContextValue } from "./context";
import { clearAuthToken } from "@/lib/client";
import { logout } from "./api";

type AuthProviderProps = {
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

export const AuthProvider: FC<AuthProviderProps> = ({
  children,
  onAuthFailure,
}) => {
  // Use useSyncExternalStore to subscribe to the WebSocket client's state
  const connectionState = useSyncExternalStore(
    subscribeToConnectionState,
    getConnectionStateSnapshot,
    getConnectionStateSnapshot, // Server snapshot (same as client for now)
  );

  // Subscribe to auth failure events
  useEffect(() => {
    if (!onAuthFailure) return;

    return wsClient.subscribeToAuthFailure(() => {
      console.log("Auth failure, redirecting to login...");
      window.localStorage.removeItem("auth");
      onAuthFailure();
    });
  }, [onAuthFailure]);

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

  const signOut = useCallback(
    () =>
      logout().then(() => {
        disconnectWebSocket();
        onAuthFailure?.();
      }),
    [onAuthFailure],
  );

  const value = useMemo<AuthContextValue>(
    () => ({
      connectionState,
      wsClient,
      reconnect,
      signOut,
    }),
    [connectionState, reconnect, signOut],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};
