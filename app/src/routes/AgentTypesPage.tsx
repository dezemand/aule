import { useState } from "react";
import { useQuery, useSpacetime } from "../hooks/useSpacetime";

function versionStatusColor(tag: string): string {
  switch (tag) {
    case "Draft":
      return "bg-gray-800 text-gray-300";
    case "Testing":
      return "bg-yellow-900/50 text-yellow-300";
    case "Active":
      return "bg-green-900/50 text-green-300";
    case "Deprecated":
      return "bg-orange-900/50 text-orange-300";
    case "Retired":
      return "bg-gray-800 text-gray-500";
    default:
      return "bg-gray-800 text-gray-400";
  }
}

function Badge({ label, color }: { label: string; color: string }) {
  return (
    <span
      className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium ${color}`}
    >
      {label}
    </span>
  );
}

export function AgentTypesPage() {
  const { conn, subscribed } = useSpacetime();
  const [showCreateType, setShowCreateType] = useState(false);
  const [showCreateVersion, setShowCreateVersion] = useState<bigint | null>(
    null
  );

  const agentTypes = useQuery((db) => Array.from(db.agent_type.iter()));
  const versions = useQuery((db) =>
    Array.from(db.agent_type_version.iter())
  );
  const runtimes = useQuery((db) => Array.from(db.agent_runtime.iter()));

  if (!subscribed) {
    return (
      <div className="flex h-full items-center justify-center text-gray-500">
        Waiting for SpacetimeDB connection...
      </div>
    );
  }

  const allTypes = agentTypes ?? [];
  const allVersions = versions ?? [];
  const allRuntimes = runtimes ?? [];

  function versionsForType(typeId: bigint) {
    return allVersions
      .filter((v) => v.agentTypeId === typeId)
      .sort(
        (a, b) =>
          b.createdAt.toDate().getTime() - a.createdAt.toDate().getTime()
      );
  }

  function runtimesForType(typeId: bigint) {
    return allRuntimes.filter(
      (r) => r.agentTypeId === typeId && r.status.tag !== "Offline"
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-gray-100">Agent Types</h1>
        <button
          onClick={() => setShowCreateType(!showCreateType)}
          className="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-blue-500"
        >
          {showCreateType ? "Cancel" : "New Agent Type"}
        </button>
      </div>

      {/* Create type form */}
      {showCreateType && conn && (
        <CreateAgentTypeForm
          conn={conn}
          onCreated={() => setShowCreateType(false)}
        />
      )}

      {/* Agent type cards */}
      {allTypes.length === 0 ? (
        <p className="text-sm text-gray-600">No agent types defined yet.</p>
      ) : (
        <div className="space-y-4">
          {allTypes.map((at) => {
            const typeVersions = versionsForType(at.id);
            const typeRuntimes = runtimesForType(at.id);

            return (
              <div
                key={Number(at.id)}
                className="rounded-lg border border-gray-800 bg-gray-900 p-4"
              >
                <div className="flex items-start justify-between">
                  <div>
                    <h3 className="font-medium text-gray-200">{at.name}</h3>
                    <p className="mt-0.5 text-sm text-gray-400">
                      {at.description}
                    </p>
                  </div>
                  <div className="flex items-center gap-3 text-xs text-gray-500">
                    <span>
                      {typeRuntimes.length} runtime
                      {typeRuntimes.length !== 1 ? "s" : ""} online
                    </span>
                    <span>
                      {typeVersions.length} version
                      {typeVersions.length !== 1 ? "s" : ""}
                    </span>
                  </div>
                </div>

                {/* Versions */}
                <div className="mt-3 border-t border-gray-800 pt-3">
                  <div className="flex items-center justify-between mb-2">
                    <p className="text-xs font-medium uppercase tracking-wider text-gray-600">
                      Versions
                    </p>
                    <button
                      onClick={() =>
                        setShowCreateVersion(
                          showCreateVersion === at.id ? null : at.id
                        )
                      }
                      className="text-xs text-blue-400 hover:text-blue-300"
                    >
                      {showCreateVersion === at.id
                        ? "Cancel"
                        : "+ Add Version"}
                    </button>
                  </div>

                  {showCreateVersion === at.id && conn && (
                    <CreateVersionForm
                      conn={conn}
                      agentTypeId={at.id}
                      onCreated={() => setShowCreateVersion(null)}
                    />
                  )}

                  {typeVersions.length === 0 ? (
                    <p className="text-xs text-gray-600">
                      No versions yet.
                    </p>
                  ) : (
                    <div className="space-y-1.5">
                      {typeVersions.map((v) => (
                        <div
                          key={Number(v.id)}
                          className="flex items-center justify-between rounded-md bg-gray-800/50 px-3 py-2 text-sm"
                        >
                          <div className="flex items-center gap-2">
                            <span className="font-mono text-gray-300">
                              {v.version}
                            </span>
                            <Badge
                              label={v.status.tag}
                              color={versionStatusColor(v.status.tag)}
                            />
                          </div>
                          <div className="flex items-center gap-2">
                            {v.status.tag === "Draft" && conn && (
                              <button
                                onClick={() =>
                                  conn.reducers.activateAgentTypeVersion({
                                    versionId: v.id,
                                  })
                                }
                                className="text-xs text-blue-400 hover:text-blue-300"
                              >
                                Activate
                              </button>
                            )}
                            <span className="text-xs text-gray-600">
                              {v.createdAt.toDate().toLocaleDateString()}
                            </span>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

// -- Create Agent Type Form --

function CreateAgentTypeForm({
  conn,
  onCreated,
}: {
  conn: NonNullable<ReturnType<typeof useSpacetime>["conn"]>;
  onCreated: () => void;
}) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name) return;

    conn.reducers.createAgentType({ name, description });
    setName("");
    setDescription("");
    onCreated();
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="rounded-lg border border-gray-800 bg-gray-900 p-4 space-y-3"
    >
      <div>
        <label className="block text-xs font-medium text-gray-400 mb-1">
          Name
        </label>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.currentTarget.value)}
          className="w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-1.5 text-sm text-gray-200"
          required
        />
      </div>
      <div>
        <label className="block text-xs font-medium text-gray-400 mb-1">
          Description
        </label>
        <textarea
          value={description}
          onChange={(e) => setDescription(e.currentTarget.value)}
          rows={2}
          className="w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-1.5 text-sm text-gray-200"
        />
      </div>
      <button
        type="submit"
        className="rounded-md bg-blue-600 px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-blue-500"
      >
        Create
      </button>
    </form>
  );
}

// -- Create Version Form --

function CreateVersionForm({
  conn,
  agentTypeId,
  onCreated,
}: {
  conn: NonNullable<ReturnType<typeof useSpacetime>["conn"]>;
  agentTypeId: bigint;
  onCreated: () => void;
}) {
  const [version, setVersion] = useState("");
  const [systemPrompt, setSystemPrompt] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!version || !systemPrompt) return;

    conn.reducers.createAgentTypeVersion({
      agentTypeId,
      version,
      systemPrompt,
    });
    setVersion("");
    setSystemPrompt("");
    onCreated();
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="mb-3 rounded-md border border-gray-700 bg-gray-800/50 p-3 space-y-2"
    >
      <div>
        <label className="block text-xs font-medium text-gray-400 mb-1">
          Version
        </label>
        <input
          type="text"
          value={version}
          onChange={(e) => setVersion(e.currentTarget.value)}
          placeholder="e.g. 0.1.0"
          className="w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-1.5 text-sm text-gray-200"
          required
        />
      </div>
      <div>
        <label className="block text-xs font-medium text-gray-400 mb-1">
          System Prompt
        </label>
        <textarea
          value={systemPrompt}
          onChange={(e) => setSystemPrompt(e.currentTarget.value)}
          rows={4}
          className="w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-1.5 text-sm text-gray-200"
          required
        />
      </div>
      <button
        type="submit"
        className="rounded-md bg-blue-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-blue-500"
      >
        Add Version
      </button>
    </form>
  );
}
