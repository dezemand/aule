type Listener = () => void;

/**
 * Microtask-coalesced event bus for bridging SpacetimeDB table events
 * to React's useSyncExternalStore. Multiple notify() calls within the
 * same microtask are batched into a single listener notification.
 */
export class TableEventBus {
  private listeners = new Set<Listener>();
  private version = 0;
  private pending = false;

  subscribe = (listener: Listener): (() => void) => {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  };

  getVersion = (): number => {
    return this.version;
  };

  notify(): void {
    if (this.pending) return;
    this.pending = true;

    queueMicrotask(() => {
      this.pending = false;
      this.version++;
      for (const listener of this.listeners) {
        listener();
      }
    });
  }
}
