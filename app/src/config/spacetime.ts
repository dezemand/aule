import { DbConnection, type ErrorContext } from "@/module_bindings";
import type { Identity } from "spacetimedb";

const SPACETIMEDB_HOST = "ws://localhost:3000";
const SPACETIMEDB_DB = "aule";
const TOKEN_KEY = "aule-spacetimedb-token";

export interface SpacetimeConfig {
  host?: string;
  databaseName?: string;
  onConnect?: (ctx: DbConnection, identity: Identity, token: string) => void;
  onConnectError?: (error: ErrorContext) => void;
  onDisconnect?: () => void;
}

export function getConnectionBuilder({
  host = SPACETIMEDB_HOST,
  databaseName = SPACETIMEDB_DB,
  onConnect,
  onConnectError,
  onDisconnect,
}: SpacetimeConfig = {}) {
  const savedToken = localStorage.getItem(TOKEN_KEY) ?? undefined;

  return DbConnection.builder()
    .withUri(host)
    .withDatabaseName(databaseName)
    .withToken(savedToken)
    .onConnect((ctx: DbConnection, identity: Identity, token: string) => {
      localStorage.setItem(TOKEN_KEY, token);
      onConnect?.(ctx, identity, token);
    })
    .onConnectError((error: ErrorContext) => {
      onConnectError?.(error);
    })
    .onDisconnect(() => {
      onDisconnect?.();
    });
}
