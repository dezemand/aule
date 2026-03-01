import { useCallback, useContext, useEffect, useState } from "react";
import type { RowTypedQuery } from "spacetimedb";
import {
  DbConnection,
  type ErrorContext,
  type SubscriptionEventContext,
} from "../module_bindings";
import {
  SpacetimeContext,
  type SpacetimeContextValue,
} from "../providers/spacetime";

export function useSpacetime(): SpacetimeContextValue {
  const ctx = useContext(SpacetimeContext);
  if (!ctx)
    throw new Error("useSpacetime must be used within SpacetimeProvider");
  return ctx;
}

interface SubscriptionState {
  version: number;
  subscribed: boolean;
  error?: string;
}

type TableDependency = {
  onInsert: (listener: () => void) => unknown;
  removeOnInsert: (listener: () => void) => unknown;
  onUpdate?: (listener: () => void) => unknown;
  removeOnUpdate?: (listener: () => void) => unknown;
  onDelete?: (listener: () => void) => unknown;
  removeOnDelete?: (listener: () => void) => unknown;
};

/**
 * Subscribes to a specific query and table dependency set.
 *
 * `query` and `dependencies` must be stable references (module-level constants,
 * `useMemo`, or `useCallback`) to avoid re-subscribing on every render.
 * See callers in `app/src/routes/AgentTypesPage.tsx`,
 * `app/src/routes/TasksPage.tsx`, and `app/src/routes/TaskDetailsPage.tsx`.
 */
export function useSubscription(
  query: (string | RowTypedQuery<unknown, unknown>)[],
  dependencies: (db: DbConnection["db"]) => TableDependency[],
): SubscriptionState {
  const { ctx } = useSpacetime();
  const [state, setState] = useState<SubscriptionState>({
    version: 0,
    subscribed: false,
  });
  const bump = useCallback(
    () => setState((s) => ({ ...s, version: s.version + 1 })),
    [],
  );

  useEffect(() => {
    if (!ctx) {
      setState((s) => ({
        ...s,
        subscribed: false,
        error: undefined,
      }));
      return;
    }

    setState((s) => ({
      ...s,
      subscribed: false,
      error: undefined,
    }));

    let cancelled = false;

    const deps = dependencies(ctx.db);
    const cleanups: Array<() => void> = [];

    for (const dep of deps) {
      dep.onInsert(bump);
      dep.onUpdate?.(bump);
      dep.onDelete?.(bump);

      cleanups.push(() => dep.removeOnInsert(bump));
      cleanups.push(() => dep.removeOnUpdate?.(bump));
      cleanups.push(() => dep.removeOnDelete?.(bump));
    }

    const subscription = ctx
      .subscriptionBuilder()
      .onApplied((_subCtx: SubscriptionEventContext) => {
        if (cancelled) return;
        setState((s) => ({ ...s, subscribed: true, error: undefined }));
      })
      .onError((_errCtx: ErrorContext) => {
        if (cancelled) return;
        setState((s) => ({
          ...s,
          error: "Subscription error",
          subscribed: false,
        }));
      })
      .subscribe(query);

    return () => {
      cancelled = true;

      for (const cleanup of cleanups) {
        cleanup();
      }

      if (subscription.isActive()) {
        subscription.unsubscribe();
      }
    };
  }, [ctx, bump, query, dependencies]);

  return state;
}

export function useQuery<T>(
  state: SubscriptionState,
  selector: (db: DbConnection["db"]) => T,
): T | undefined {
  const { ctx } = useSpacetime();
  const { subscribed } = state;
  // Re-renders come from parent `bump` incrementing `state.version`; this hook
  // intentionally only gates access on `ctx` and `subscribed`.
  if (!ctx || !subscribed) return undefined;
  return selector(ctx.db);
}
