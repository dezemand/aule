import type {
  DbConnection,
  SubscriptionEventContext,
  ErrorContext,
  SubscriptionHandle,
} from "@/module_bindings";
import { TableEventBus } from "./tableEventBus";
import type { SubscriptionDef } from "./subscription";

type EntryStatus = "pending" | "active" | "grace" | "ended";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyCallback = (...args: any[]) => void;

type TableHandleWithEvents = {
  onInsert?: (cb: AnyCallback) => void;
  removeOnInsert?: (cb: AnyCallback) => void;
  onUpdate?: (cb: AnyCallback) => void;
  removeOnUpdate?: (cb: AnyCallback) => void;
  onDelete?: (cb: AnyCallback) => void;
  removeOnDelete?: (cb: AnyCallback) => void;
};

type SubscriptionEntry = {
  key: string;
  def: SubscriptionDef;
  status: EntryStatus;
  stdbHandle: SubscriptionHandle | null;
  bus: TableEventBus;
  refCount: number;
  graceTimeout: ReturnType<typeof setTimeout> | null;
  readyPromise: Promise<void>;
  readyResolve: () => void;
  readyReject: (err: Error) => void;
  cleanups: (() => void)[];
  error: string | null;
};

type SubscriptionManagerOptions = {
  graceTTL?: number;
};

const DEFAULT_GRACE_TTL = 30_000;

export class SubscriptionManager {
  private entries = new Map<string, SubscriptionEntry>();
  private conn: DbConnection | null = null;
  private connPromise: Promise<DbConnection>;
  private connResolve!: (conn: DbConnection) => void;
  private graceTTL: number;

  constructor(opts: SubscriptionManagerOptions = {}) {
    this.graceTTL = opts.graceTTL ?? DEFAULT_GRACE_TTL;
    this.connPromise = new Promise((resolve) => {
      this.connResolve = resolve;
    });
  }

  setConnection(conn: DbConnection): void {
    this.conn = conn;
    this.connResolve(conn);
  }

  clearConnection(): void {
    for (const entry of this.entries.values()) {
      entry.status = "ended";
      entry.cleanups.forEach((fn) => fn());
      entry.cleanups = [];
      if (entry.graceTimeout) {
        clearTimeout(entry.graceTimeout);
        entry.graceTimeout = null;
      }
    }
    this.entries.clear();

    this.conn = null;
    this.connPromise = new Promise((resolve) => {
      this.connResolve = resolve;
    });
  }

  async ensure(key: string, def: SubscriptionDef): Promise<void> {
    const existing = this.entries.get(key);

    if (existing) {
      if (existing.status === "ended" || existing.stdbHandle?.isEnded()) {
        this.cleanup(key);
      } else {
        if (existing.status === "grace" && existing.graceTimeout) {
          clearTimeout(existing.graceTimeout);
          existing.graceTimeout = null;
          existing.status = "active";
        }
        await existing.readyPromise;
        return;
      }
    }

    const conn = this.conn ?? (await this.connPromise);

    let readyResolve!: () => void;
    let readyReject!: (err: Error) => void;
    const readyPromise = new Promise<void>((resolve, reject) => {
      readyResolve = resolve;
      readyReject = reject;
    });

    const bus = new TableEventBus();

    const entry: SubscriptionEntry = {
      key,
      def,
      status: "pending",
      stdbHandle: null,
      bus,
      refCount: 0,
      graceTimeout: null,
      readyPromise,
      readyResolve,
      readyReject,
      cleanups: [],
      error: null,
    };

    this.entries.set(key, entry);

    const subscription = conn
      .subscriptionBuilder()
      .onApplied((_ctx: SubscriptionEventContext) => {
        entry.status = "active";
        entry.readyResolve();
        entry.bus.notify();
      })
      .onError((_ctx: ErrorContext) => {
        console.error(`[SubscriptionManager] "${key}" subscription error`);
        entry.status = "ended";
        entry.error = `Subscription "${key}" failed`;
        entry.readyResolve();
        entry.bus.notify();
      })
      .subscribe(def.query);

    entry.stdbHandle = subscription;

    this.wireTableEvents(conn, entry);
  }

  retain(key: string): void {
    const entry = this.entries.get(key);
    if (!entry) return;

    entry.refCount++;

    if (entry.status === "grace" && entry.graceTimeout) {
      clearTimeout(entry.graceTimeout);
      entry.graceTimeout = null;
      entry.status = "active";
    }
  }

  release(key: string): void {
    const entry = this.entries.get(key);
    if (!entry) return;

    entry.refCount = Math.max(0, entry.refCount - 1);

    if (entry.refCount === 0 && entry.status === "active") {
      entry.status = "grace";
      entry.graceTimeout = setTimeout(() => {
        this.teardown(key);
      }, this.graceTTL);
    }
  }

  getBus(key: string): TableEventBus | undefined {
    return this.entries.get(key)?.bus;
  }

  getStatus(key: string): EntryStatus | undefined {
    const entry = this.entries.get(key);
    if (!entry) return undefined;
    if (entry.stdbHandle?.isEnded() && entry.status !== "ended") {
      entry.status = "ended";
    }
    return entry.status;
  }

  getError(key: string): string | null {
    return this.entries.get(key)?.error ?? null;
  }

  getConnection(): DbConnection | null {
    return this.conn;
  }

  waitForConnection(): Promise<DbConnection> {
    return this.conn ? Promise.resolve(this.conn) : this.connPromise;
  }

  private wireTableEvents(conn: DbConnection, entry: SubscriptionEntry): void {
    const db = conn.db as unknown as Record<string, TableHandleWithEvents>;

    for (const tableName of entry.def.tables) {
      const table = db[tableName];
      if (!table) continue;

      if (table.onInsert && table.removeOnInsert) {
        const cb: AnyCallback = () => entry.bus.notify();
        table.onInsert(cb);
        entry.cleanups.push(() => table.removeOnInsert!(cb));
      }

      if (table.onUpdate && table.removeOnUpdate) {
        const cb: AnyCallback = () => entry.bus.notify();
        table.onUpdate(cb);
        entry.cleanups.push(() => table.removeOnUpdate!(cb));
      }

      if (table.onDelete && table.removeOnDelete) {
        const cb: AnyCallback = () => entry.bus.notify();
        table.onDelete(cb);
        entry.cleanups.push(() => table.removeOnDelete!(cb));
      }
    }
  }

  private teardown(key: string): void {
    const entry = this.entries.get(key);
    if (!entry) return;

    entry.cleanups.forEach((fn) => fn());
    entry.cleanups = [];

    if (entry.stdbHandle && !entry.stdbHandle.isEnded()) {
      entry.stdbHandle.unsubscribeThen(() => {
        entry.status = "ended";
        this.entries.delete(key);
      });
    } else {
      entry.status = "ended";
      this.entries.delete(key);
    }
  }

  private cleanup(key: string): void {
    const entry = this.entries.get(key);
    if (!entry) return;

    entry.cleanups.forEach((fn) => fn());
    entry.cleanups = [];
    if (entry.graceTimeout) {
      clearTimeout(entry.graceTimeout);
    }
    this.entries.delete(key);
  }
}
