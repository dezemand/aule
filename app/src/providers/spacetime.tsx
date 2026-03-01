import type { Identity } from "spacetimedb";
import { DbConnection, type ErrorContext } from "../module_bindings";
import { createContext, useEffect, useState, type ReactNode } from "react";

export interface SpacetimeContextValue {
  ctx: DbConnection | null;
  identity: Identity | null;
  connected: boolean;
  error: string | null;
}

export const SpacetimeContext = createContext<SpacetimeContextValue | null>(
  null,
);

const SPACETIMEDB_HOST = "ws://localhost:3000";
const SPACETIMEDB_DB = "aule";
const TOKEN_KEY = "aule-spacetimedb-token";

interface SpacetimeProviderProps {
  children: ReactNode;
  host?: string;
  databaseName?: string;
}

export function SpacetimeProvider({
  children,
  host = SPACETIMEDB_HOST,
  databaseName = SPACETIMEDB_DB,
}: SpacetimeProviderProps) {
  const [state, setState] = useState<SpacetimeContextValue>({
    ctx: null,
    identity: null,
    connected: false,
    error: null,
  });

  useEffect(() => {
    let cancelled = false;
    const savedToken = localStorage.getItem(TOKEN_KEY) ?? undefined;

    const conn = DbConnection.builder()
      .withUri(host)
      .withDatabaseName(databaseName)
      .withToken(savedToken)
      .onConnect((ctx: DbConnection, identity: Identity, token: string) => {
        if (cancelled) return;
        localStorage.setItem(TOKEN_KEY, token);
        setState((s) => ({
          ...s,
          ctx,
          identity,
          connected: true,
          error: null,
        }));
      })
      .onConnectError((_ctx: ErrorContext) => {
        if (cancelled) return;
        setState((s) => ({
          ...s,
          ctx: null,
          identity: null,
          connected: false,
          error: "Failed to connect to SpacetimeDB",
        }));
      })
      .onDisconnect(() => {
        if (cancelled) return;
        setState((s) => ({
          ...s,
          ctx: null,
          identity: null,
          connected: false,
          error: null,
        }));
      })
      .build();

    setState((s) => ({ ...s, ctx: conn }));

    return () => {
      cancelled = true;
      conn.disconnect();
    };
  }, [host, databaseName]);

  return (
    <SpacetimeContext.Provider value={state}>
      {children}
    </SpacetimeContext.Provider>
  );
}
