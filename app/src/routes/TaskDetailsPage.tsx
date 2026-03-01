import { Link, useParams } from "@tanstack/react-router";
import { useCallback, useEffect, useState } from "react";
import { Tabs } from "@base-ui-components/react/tabs";
import { Badge } from "../components/Badge";
import { ConversationView } from "../components/ConversationView";
import { Markdown } from "../components/Markdown";
import { TaskAssignmentControl } from "../components/TaskAssignmentControl";
import { useQuery, useSpacetime, useSubscription } from "../hooks/useSpacetime";
import { tables } from "../module_bindings";
import { taskStatusColor } from "../utils/statusColors";

const QUERY = [
  tables.agent_task,
  tables.agent_type,
  tables.agent_type_version,
  tables.observation,
  tables.runtime_event,
  tables.agent_runtime,
];

function formatDuration(milliseconds: number): string {
  const totalSeconds = Math.max(0, Math.floor(milliseconds / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  const hours = Math.floor(minutes / 60);

  if (hours > 0) {
    return `${hours}h ${minutes % 60}m ${seconds}s`;
  }
  if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  }
  return `${seconds}s`;
}

function formatTimestamp(date: Date): string {
  return date.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

// ─── Metadata sidebar row ───────────────────────────────────────────

function Property({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-start justify-between gap-4 py-2">
      <span className="shrink-0 text-xs text-gray-500">{label}</span>
      <span className="text-right text-xs text-gray-200">{children}</span>
    </div>
  );
}

// ─── Page ───────────────────────────────────────────────────────────

export function TaskDetailsPage() {
  const { taskId } = useParams({ from: "/tasks/$taskId" });
  const { ctx } = useSpacetime();
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, []);

  const sub = useSubscription(
    QUERY,
    useCallback(
      (db) => [
        db.agent_task,
        db.agent_type,
        db.agent_type_version,
        db.observation,
        db.runtime_event,
        db.agent_runtime,
      ],
      [],
    ),
  );
  const tasks = useQuery(sub, (db) => Array.from(db.agent_task.iter()));
  const agentTypes = useQuery(sub, (db) => Array.from(db.agent_type.iter()));
  const agentTypeVersions = useQuery(sub, (db) =>
    Array.from(db.agent_type_version.iter()),
  );
  const observations = useQuery(sub, (db) => Array.from(db.observation.iter()));
  const runtimeEvents = useQuery(sub, (db) =>
    Array.from(db.runtime_event.iter()),
  );
  const runtimes = useQuery(sub, (db) => Array.from(db.agent_runtime.iter()));

  if (!sub.subscribed) {
    return (
      <div className="flex h-full items-center justify-center text-gray-500">
        Waiting for SpacetimeDB connection...
      </div>
    );
  }

  let parsedTaskId: bigint;
  try {
    parsedTaskId = BigInt(taskId);
  } catch {
    return (
      <div className="space-y-4">
        <nav className="flex items-center gap-1.5 text-xs text-gray-500">
          <Link to="/" className="hover:text-gray-300">Dashboard</Link>
          <span>/</span>
          <Link to="/tasks" className="hover:text-gray-300">Tasks</Link>
          <span>/</span>
          <span className="text-gray-400">{taskId}</span>
        </nav>
        <p className="text-sm text-red-400">Invalid task id: {taskId}</p>
      </div>
    );
  }

  const task = (tasks ?? []).find((c) => c.id === parsedTaskId);
  const agentTypeMap = new Map(
    (agentTypes ?? []).map((a) => [Number(a.id), a.name]),
  );
  const runtimeNameMap = new Map(
    (runtimes ?? []).map((r) => [r.identity.toHexString(), r.name]),
  );

  if (!task) {
    return (
      <div className="space-y-4">
        <nav className="flex items-center gap-1.5 text-xs text-gray-500">
          <Link to="/" className="hover:text-gray-300">Dashboard</Link>
          <span>/</span>
          <Link to="/tasks" className="hover:text-gray-300">Tasks</Link>
          <span>/</span>
          <span className="text-gray-400">#{taskId}</span>
        </nav>
        <p className="text-sm text-gray-500">Task #{taskId} was not found.</p>
      </div>
    );
  }

  // ── Derived data ────────────────────────────────────────────────

  const taskObservations = (observations ?? [])
    .filter((o) => o.taskId === task.id)
    .sort(
      (a, b) => a.createdAt.toDate().getTime() - b.createdAt.toDate().getTime(),
    );

  const taskRuntimeEvents = (runtimeEvents ?? [])
    .filter((e) => e.taskId === task.id)
    .sort(
      (a, b) => a.createdAt.toDate().getTime() - b.createdAt.toDate().getTime(),
    );

  const latestTurn = taskRuntimeEvents.reduce(
    (max, e) => Math.max(max, Number(e.turn)),
    0,
  );

  const activeVersion = (agentTypeVersions ?? []).find(
    (v) => v.agentTypeId === task.agentTypeId && v.status.tag === "Active",
  );

  const isRunning = task.status.tag === "Running";
  const isTerminal =
    task.status.tag === "Completed" ||
    task.status.tag === "Failed" ||
    task.status.tag === "Cancelled";

  const elapsed = task.startedAt
    ? formatDuration(
        (isTerminal && task.completedAt
          ? task.completedAt.toDate().getTime()
          : now) - task.startedAt.toDate().getTime(),
      )
    : null;

  const assignedRuntimeName = task.assignedRuntime
    ? (runtimeNameMap.get(task.assignedRuntime.toHexString()) ?? "Unknown")
    : null;

  const agentTypeName =
    agentTypeMap.get(Number(task.agentTypeId)) ?? "Unknown";

  // ── Render ──────────────────────────────────────────────────────

  return (
    <div className="flex h-full flex-col">
      {/* ── Running progress bar ─────────────────────────────────── */}
      {isRunning && (
        <div className="h-0.5 w-full overflow-hidden bg-gray-800">
          <div
            className="h-full bg-yellow-400/80 transition-all duration-500"
            style={{ width: `${Math.min((latestTurn / 24) * 100, 100)}%` }}
          />
        </div>
      )}

      {/* ── Header ───────────────────────────────────────────────── */}
      <header className="space-y-3 pb-4">
        <nav className="flex items-center gap-1.5 text-xs text-gray-500">
          <Link to="/" className="hover:text-gray-300">
            Dashboard
          </Link>
          <span>/</span>
          <Link to="/tasks" className="hover:text-gray-300">
            Tasks
          </Link>
          <span>/</span>
          <span className="text-gray-400">#{Number(task.id)}</span>
        </nav>

        <div className="flex items-center gap-3">
          <h1 className="text-lg font-semibold text-gray-100">{task.title}</h1>
          <Badge
            label={task.status.tag}
            color={taskStatusColor(task.status.tag)}
          />
          {isRunning && elapsed && (
            <span className="text-xs text-yellow-300/80">{elapsed}</span>
          )}
          <span className="ml-auto text-xs text-gray-600">
            #{Number(task.id)}
          </span>
        </div>
      </header>

      {/* ── Tabs ─────────────────────────────────────────────────── */}
      <Tabs.Root defaultValue="conversation" className="flex min-h-0 flex-1 flex-col">
        <Tabs.List className="flex shrink-0 border-b border-gray-800">
          <Tabs.Tab
            value="conversation"
            className="border-b-2 border-transparent px-4 py-2 text-sm text-gray-400 hover:text-gray-200 data-[active]:border-gray-100 data-[active]:text-gray-100"
          >
            Conversation
          </Tabs.Tab>
          <Tabs.Tab
            value="details"
            className="border-b-2 border-transparent px-4 py-2 text-sm text-gray-400 hover:text-gray-200 data-[active]:border-gray-100 data-[active]:text-gray-100"
          >
            Details
          </Tabs.Tab>
          <Tabs.Tab
            value="observations"
            className="border-b-2 border-transparent px-4 py-2 text-sm text-gray-400 hover:text-gray-200 data-[active]:border-gray-100 data-[active]:text-gray-100"
          >
            Observations
            {taskObservations.length > 0 && (
              <span className="ml-1.5 rounded-full bg-gray-800 px-1.5 py-0.5 text-xs text-gray-400">
                {taskObservations.length}
              </span>
            )}
          </Tabs.Tab>
        </Tabs.List>

        {/* ── Conversation ─────────────────────────────────────── */}
        <Tabs.Panel
          value="conversation"
          className="min-h-0 flex-1 overflow-y-auto pt-4 outline-none"
        >
          <ConversationView
            events={taskRuntimeEvents}
            taskDescription={task.description}
            taskStatus={task.status.tag}
            systemPrompt={activeVersion?.systemPrompt}
          />
        </Tabs.Panel>

        {/* ── Details ──────────────────────────────────────────── */}
        <Tabs.Panel
          value="details"
          className="min-h-0 flex-1 overflow-y-auto pt-4 outline-none"
        >
          <div className="grid gap-6 lg:grid-cols-[1fr_280px]">
            {/* Left: description + result */}
            <div className="space-y-6">
              <section>
                <h2 className="mb-3 text-xs font-medium uppercase tracking-wider text-gray-500">
                  Description
                </h2>
                <div className="rounded-lg border border-gray-800 bg-gray-900 p-5">
                  <Markdown className="text-sm leading-relaxed text-gray-200">
                    {task.description || "No description provided."}
                  </Markdown>
                </div>
              </section>

              {task.result && (
                <section>
                  <h2 className="mb-3 text-xs font-medium uppercase tracking-wider text-gray-500">
                    Result
                  </h2>
                  <div
                    className={`rounded-lg border p-5 ${
                      task.status.tag === "Failed"
                        ? "border-red-900/50 bg-red-950/20"
                        : "border-green-900/50 bg-green-950/20"
                    }`}
                  >
                    <Markdown
                      className={`text-sm leading-relaxed ${
                        task.status.tag === "Failed"
                          ? "text-red-100"
                          : "text-green-100"
                      }`}
                    >
                      {task.result}
                    </Markdown>
                  </div>
                </section>
              )}

              {activeVersion && (
                <section>
                  <h2 className="mb-3 text-xs font-medium uppercase tracking-wider text-gray-500">
                    System prompt
                  </h2>
                  <details className="rounded-lg border border-gray-800 bg-gray-900">
                    <summary className="cursor-pointer px-5 py-3 text-xs text-gray-400 hover:text-gray-200">
                      v{activeVersion.version} — click to expand
                    </summary>
                    <div className="border-t border-gray-800 px-5 py-4">
                      <Markdown className="text-sm text-gray-300">
                        {activeVersion.systemPrompt}
                      </Markdown>
                    </div>
                  </details>
                </section>
              )}
            </div>

            {/* Right: property sidebar */}
            <aside className="space-y-5">
              <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
                <h3 className="mb-2 text-xs font-medium uppercase tracking-wider text-gray-500">
                  Properties
                </h3>
                <div className="divide-y divide-gray-800">
                  <Property label="Status">
                    <Badge
                      label={task.status.tag}
                      color={taskStatusColor(task.status.tag)}
                    />
                  </Property>
                  <Property label="Agent type">{agentTypeName}</Property>
                  <Property label="Runtime">
                    {assignedRuntimeName ?? (
                      <span className="text-gray-500">Unassigned</span>
                    )}
                  </Property>
                  <Property label="Turn">
                    {latestTurn > 0 ? (
                      `${latestTurn} / 24`
                    ) : (
                      <span className="text-gray-500">—</span>
                    )}
                  </Property>
                  {elapsed && (
                    <Property label="Duration">{elapsed}</Property>
                  )}
                  <Property label="Created">
                    {formatTimestamp(task.createdAt.toDate())}
                  </Property>
                  {task.startedAt && (
                    <Property label="Started">
                      {formatTimestamp(task.startedAt.toDate())}
                    </Property>
                  )}
                  {task.completedAt && (
                    <Property label="Completed">
                      {formatTimestamp(task.completedAt.toDate())}
                    </Property>
                  )}
                </div>
              </div>

              {task.status.tag === "Pending" && (
                <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
                  <h3 className="mb-3 text-xs font-medium uppercase tracking-wider text-gray-500">
                    Assignment
                  </h3>
                  <TaskAssignmentControl
                    taskId={task.id}
                    taskStatus={task.status.tag}
                    assignedRuntime={task.assignedRuntime ?? null}
                    runtimes={runtimes ?? []}
                    conn={ctx}
                    compact
                  />
                </div>
              )}
            </aside>
          </div>
        </Tabs.Panel>

        {/* ── Observations ─────────────────────────────────────── */}
        <Tabs.Panel
          value="observations"
          className="min-h-0 flex-1 overflow-y-auto pt-4 outline-none"
        >
          {taskObservations.length === 0 ? (
            <p className="text-sm text-gray-600">No observations yet.</p>
          ) : (
            <div className="space-y-2">
              {taskObservations.map((obs) => (
                <div
                  key={Number(obs.id)}
                  className="rounded-lg border border-gray-800 bg-gray-900 px-4 py-3"
                >
                  <div className="flex items-center gap-2 text-xs text-gray-500">
                    <Badge
                      label={obs.kind.tag}
                      color={
                        obs.kind.tag === "Error"
                          ? "bg-red-900/50 text-red-300"
                          : obs.kind.tag === "Result"
                            ? "bg-green-900/50 text-green-300"
                            : "bg-gray-800 text-gray-300"
                      }
                    />
                    <span>
                      {formatTimestamp(obs.createdAt.toDate())}
                    </span>
                  </div>
                  <Markdown className="mt-1 text-sm text-gray-300">
                    {obs.content}
                  </Markdown>
                </div>
              ))}
            </div>
          )}
        </Tabs.Panel>
      </Tabs.Root>
    </div>
  );
}
