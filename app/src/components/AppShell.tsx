import { Link, useRouterState } from "@tanstack/react-router";
import { useSpacetime } from "../hooks/useSpacetime";
import type { ReactNode } from "react";

const NAV_ITEMS = [
  { to: "/", label: "Dashboard" },
  { to: "/tasks", label: "Tasks" },
  { to: "/agent-types", label: "Agent Types" },
] as const;

function ConnectionBadge() {
  const { connected, subscribed, error } = useSpacetime();

  if (error) {
    return (
      <span className="inline-flex items-center gap-1.5 text-xs text-red-400">
        <span className="h-2 w-2 rounded-full bg-red-500" />
        {error}
      </span>
    );
  }

  if (!connected) {
    return (
      <span className="inline-flex items-center gap-1.5 text-xs text-gray-500">
        <span className="h-2 w-2 rounded-full bg-gray-600" />
        Disconnected
      </span>
    );
  }

  if (!subscribed) {
    return (
      <span className="inline-flex items-center gap-1.5 text-xs text-yellow-400">
        <span className="h-2 w-2 rounded-full bg-yellow-500 animate-pulse" />
        Connecting...
      </span>
    );
  }

  return (
    <span className="inline-flex items-center gap-1.5 text-xs text-green-400">
      <span className="h-2 w-2 rounded-full bg-green-500" />
      Connected
    </span>
  );
}

export function AppShell({ children }: { children: ReactNode }) {
  const routerState = useRouterState();
  const currentPath = routerState.location.pathname;

  return (
    <div className="flex h-screen">
      {/* Sidebar */}
      <aside className="flex w-56 flex-col border-r border-gray-800 bg-gray-900">
        <div className="flex h-14 items-center gap-2 border-b border-gray-800 px-4">
          <span className="text-lg font-semibold tracking-tight text-gray-100">
            Aule
          </span>
          <span className="text-xs text-gray-500">dashboard</span>
        </div>

        <nav className="flex-1 space-y-0.5 p-2">
          {NAV_ITEMS.map((item) => {
            const active = currentPath === item.to;
            return (
              <Link
                key={item.to}
                to={item.to}
                className={`block rounded-md px-3 py-2 text-sm transition-colors ${
                  active
                    ? "bg-gray-800 text-gray-100"
                    : "text-gray-400 hover:bg-gray-800/50 hover:text-gray-200"
                }`}
              >
                {item.label}
              </Link>
            );
          })}
        </nav>

        <div className="border-t border-gray-800 p-3">
          <ConnectionBadge />
        </div>
      </aside>

      {/* Main */}
      <main className="flex-1 overflow-y-auto p-6">{children}</main>
    </div>
  );
}
