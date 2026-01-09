import { type ConnectionState } from "../websocket/websocket-client";
import {
  useCallback,
  useEffect,
  useMemo,
  useSyncExternalStore,
  type FC,
  type ReactNode,
} from "react";
import { disconnectWebSocket } from "../websocket/client";
import { logout } from "./api";
import { AuthContext, type AuthContextValue } from "./context";
import { useAuthStore } from "./store";
import { WsProvider } from "../websocket/provider";

type AuthProviderProps = {
  children: ReactNode;
  /**
   * Called when authentication fails and user should be redirected to login.
   */
  onAuthFailure?: () => void;
};

export const AuthProvider: FC<AuthProviderProps> = ({
  children,
  onAuthFailure,
}) => {
  const { claims } = useAuthStore();
  const signOut = useCallback(
    () =>
      logout().then(() => {
        disconnectWebSocket();
        onAuthFailure?.();
      }),
    [onAuthFailure],
  );
  const value = useMemo<AuthContextValue>(
    () => ({ signOut, claims }),
    [signOut, claims],
  );

  if (!claims) {
    return (
      <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
    );
  }

  return (
    <AuthContext.Provider value={value}>
      <WsProvider>{children}</WsProvider>
    </AuthContext.Provider>
  );
};
