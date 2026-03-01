import {
  createContext,
  useContext,
  useEffect,
  useState,
  useCallback,
  type ReactNode,
} from "react";
import {
  DbConnection,
  type EventContext,
  type ErrorContext,
  type SubscriptionEventContext,
} from "../module_bindings";
import type { Identity } from "spacetimedb";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface SpacetimeState {
  conn: DbConnection | null;
  identity: Identity | null;
  connected: boolean;
  subscribed: boolean;
  error: string | null;
}

interface SpacetimeContextValue extends SpacetimeState {
  /** Monotonically increasing counter bumped on every DB event. */
  version: number;
}

// ---------------------------------------------------------------------------
// Context
// ---------------------------------------------------------------------------

const SpacetimeContext = createContext<SpacetimeContextValue | null>(null);

const SPACETIMEDB_HOST = "ws://localhost:3000";
const SPACETIMEDB_DB = "aule";
const TOKEN_KEY = "aule-spacetimedb-token";

// ---------------------------------------------------------------------------
// Provider
// ---------------------------------------------------------------------------

export function SpacetimeProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<SpacetimeState>({
    conn: null,
    identity: null,
    connected: false,
    subscribed: false,
    error: null,
  });

  // Version counter that bumps on every table event so consumers re-render.
  const [version, setVersion] = useState(0);
  const bump = useCallback(() => setVersion((v) => v + 1), []);

  useEffect(() => {
    let cancelled = false;
    const savedToken = localStorage.getItem(TOKEN_KEY) ?? undefined;

    const conn = DbConnection.builder()
      .withUri(SPACETIMEDB_HOST)
      .withDatabaseName(SPACETIMEDB_DB)
      .withToken(savedToken)
      .onConnect((ctx: DbConnection, identity: Identity, token: string) => {
        if (cancelled) return;
        localStorage.setItem(TOKEN_KEY, token);
        setState((s) => ({ ...s, conn: ctx, identity, connected: true }));

        // Register bump callbacks on all tables
        ctx.db.agent_runtime.onInsert(() => bump());
        ctx.db.agent_runtime.onUpdate(() => bump());
        ctx.db.agent_runtime.onDelete(() => bump());
        ctx.db.agent_task.onInsert(() => bump());
        ctx.db.agent_task.onUpdate(() => bump());
        ctx.db.agent_task.onDelete(() => bump());
        ctx.db.agent_type.onInsert(() => bump());
        ctx.db.agent_type.onUpdate(() => bump());
        ctx.db.agent_type.onDelete(() => bump());
        ctx.db.agent_type_version.onInsert(() => bump());
        ctx.db.agent_type_version.onUpdate(() => bump());
        ctx.db.agent_type_version.onDelete(() => bump());
        ctx.db.observation.onInsert(() => bump());
        ctx.db.observation.onUpdate(() => bump());
        ctx.db.observation.onDelete(() => bump());
        ctx.db.runtime_event.onInsert(() => bump());
        ctx.db.runtime_event.onUpdate(() => bump());
        ctx.db.runtime_event.onDelete(() => bump());

        // Subscribe to all tables
        ctx
          .subscriptionBuilder()
          .onApplied((_subCtx: SubscriptionEventContext) => {
            if (cancelled) return;
            setState((s) => ({ ...s, subscribed: true }));
            bump();
          })
          .onError((_errCtx: ErrorContext) => {
            if (cancelled) return;
            setState((s) => ({ ...s, error: "Subscription error" }));
          })
          .subscribe([
            "SELECT * FROM agent_runtime",
            "SELECT * FROM agent_task",
            "SELECT * FROM agent_type",
            "SELECT * FROM agent_type_version",
            "SELECT * FROM observation",
            "SELECT * FROM runtime_event",
          ]);
      })
      .onConnectError((_ctx: ErrorContext) => {
        if (cancelled) return;
        setState((s) => ({
          ...s,
          error: "Failed to connect to SpacetimeDB",
        }));
      })
      .onDisconnect(() => {
        if (cancelled) return;
        setState((s) => ({
          ...s,
          connected: false,
          subscribed: false,
        }));
      })
      .build();

    setState((s) => ({ ...s, conn }));

    return () => {
      cancelled = true;
      conn.disconnect();
    };
  }, [bump]);

  return (
    <SpacetimeContext.Provider value={{ ...state, version }}>
      {children}
    </SpacetimeContext.Provider>
  );
}

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------

export function useSpacetime(): SpacetimeContextValue {
  const ctx = useContext(SpacetimeContext);
  if (!ctx) throw new Error("useSpacetime must be used within SpacetimeProvider");
  return ctx;
}

/**
 * Read rows from a SpacetimeDB table. Re-renders when any table changes.
 * The selector receives `conn.db` and should return the data you need.
 */
export function useQuery<T>(selector: (db: DbConnection["db"]) => T): T | undefined {
  const { conn, subscribed, version } = useSpacetime();
  if (!conn || !subscribed) return undefined;
  // version is in the dependency to ensure fresh reads
  return selector(conn.db);
}
