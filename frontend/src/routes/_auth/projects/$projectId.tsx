import { createFileRoute, Link } from "@tanstack/react-router";
import type { FC } from "react";
import { useSubscription } from "@/services/subscriptions/use-subscription";
import { queryKeys } from "@/lib/query";
import type { Project, ProjectResponse, MembersListResponse } from "@/model/ws";

const StatusBadge: FC<{ status: Project["status"] }> = ({ status }) => {
  const colors = {
    active: "bg-green-100 text-green-800",
    paused: "bg-yellow-100 text-yellow-800",
    archived: "bg-gray-100 text-gray-800",
  };

  return (
    <span
      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-sm font-medium ${colors[status]}`}
    >
      {status}
    </span>
  );
};

const SectionCard: FC<{
  title: string;
  children: React.ReactNode;
  empty?: string;
}> = ({ title, children, empty }) => {
  const isEmpty =
    !children || (Array.isArray(children) && children.length === 0);

  return (
    <div className="bg-white border rounded-lg p-4">
      <h3 className="font-semibold text-gray-900 mb-3">{title}</h3>
      {isEmpty && empty ? (
        <p className="text-gray-400 text-sm italic">{empty}</p>
      ) : (
        children
      )}
    </div>
  );
};

const ListItems: FC<{ items?: string[] }> = ({ items }) => {
  if (!items || items.length === 0) return null;

  return (
    <ul className="list-disc list-inside space-y-1 text-sm text-gray-600">
      {items.map((item, i) => (
        <li key={i}>{item}</li>
      ))}
    </ul>
  );
};

const ProjectDetail: FC = () => {
  const { projectId } = Route.useParams();
  const { data: projectData, isLoading } = useSubscription<ProjectResponse>({
    queryKey: queryKeys.projects.detail(projectId),
    topic: "projects.detail",
    query: { project_id: projectId },
  });
  const { data: membersData, isLoading: membersLoading } =
    useSubscription<MembersListResponse>({
      queryKey: queryKeys.projects.members(projectId),
      topic: "projects.members",
      query: { project_id: projectId },
    });

  const project = projectData?.payload?.project;
  const members = membersData?.payload?.members ?? [];

  if (isLoading) {
    return (
      <div className="p-6 max-w-4xl mx-auto">
        <div className="animate-pulse space-y-4">
          <div className="h-8 bg-gray-200 rounded w-1/3"></div>
          <div className="h-4 bg-gray-200 rounded w-2/3"></div>
          <div className="h-64 bg-gray-200 rounded"></div>
        </div>
      </div>
    );
  }

  if (!project) {
    return (
      <div className="p-6 max-w-4xl mx-auto">
        <div className="bg-red-50 border border-red-200 rounded-lg p-6 text-center">
          <p className="text-red-600 font-medium">Project not found</p>
          <Link
            to="/projects"
            className="text-sm text-red-500 hover:underline mt-2 inline-block"
          >
            Back to Projects
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-4xl mx-auto">
      {/* Header */}
      <div className="mb-6">
        <Link
          to="/projects"
          className="text-sm text-gray-500 hover:text-gray-700 mb-2 inline-block"
        >
          &larr; Back to Projects
        </Link>

        <div className="flex justify-between items-start">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">{project.name}</h1>
            <span className="text-sm text-gray-400 font-mono">
              {project.key}
            </span>
          </div>
          <StatusBadge status={project.status} />
        </div>

        {project.description && (
          <p className="text-gray-600 mt-2">{project.description}</p>
        )}
      </div>

      {/* Main Grid */}
      <div className="grid gap-6 md:grid-cols-2">
        {/* Purpose & Intent */}
        <SectionCard title="Purpose & Intent" empty="No purpose defined yet">
          {project.purpose && (
            <div className="space-y-3">
              {project.purpose.goal && (
                <div>
                  <dt className="text-xs text-gray-500 uppercase">Goal</dt>
                  <dd className="text-sm text-gray-800">
                    {project.purpose.goal}
                  </dd>
                </div>
              )}
              {project.purpose.problem_statement && (
                <div>
                  <dt className="text-xs text-gray-500 uppercase">
                    Problem Statement
                  </dt>
                  <dd className="text-sm text-gray-800">
                    {project.purpose.problem_statement}
                  </dd>
                </div>
              )}
              {project.purpose.expected_value && (
                <div>
                  <dt className="text-xs text-gray-500 uppercase">
                    Expected Value
                  </dt>
                  <dd className="text-sm text-gray-800">
                    {project.purpose.expected_value}
                  </dd>
                </div>
              )}
              {project.purpose.time_horizon && (
                <div>
                  <dt className="text-xs text-gray-500 uppercase">
                    Time Horizon
                  </dt>
                  <dd className="text-sm text-gray-800">
                    {project.purpose.time_horizon}
                  </dd>
                </div>
              )}
              {project.purpose.non_goals &&
                project.purpose.non_goals.length > 0 && (
                  <div>
                    <dt className="text-xs text-gray-500 uppercase mb-1">
                      Non-Goals
                    </dt>
                    <ListItems items={project.purpose.non_goals} />
                  </div>
                )}
            </div>
          )}
        </SectionCard>

        {/* Scope & Boundaries */}
        <SectionCard title="Scope & Boundaries" empty="No scope defined yet">
          {project.scope && (
            <div className="space-y-3">
              {project.scope.in_scope && project.scope.in_scope.length > 0 && (
                <div>
                  <dt className="text-xs text-gray-500 uppercase mb-1">
                    In Scope
                  </dt>
                  <ListItems items={project.scope.in_scope} />
                </div>
              )}
              {project.scope.out_of_scope &&
                project.scope.out_of_scope.length > 0 && (
                  <div>
                    <dt className="text-xs text-gray-500 uppercase mb-1">
                      Out of Scope
                    </dt>
                    <ListItems items={project.scope.out_of_scope} />
                  </div>
                )}
              {project.scope.assumptions &&
                project.scope.assumptions.length > 0 && (
                  <div>
                    <dt className="text-xs text-gray-500 uppercase mb-1">
                      Assumptions
                    </dt>
                    <ListItems items={project.scope.assumptions} />
                  </div>
                )}
              {project.scope.constraints &&
                project.scope.constraints.length > 0 && (
                  <div>
                    <dt className="text-xs text-gray-500 uppercase mb-1">
                      Constraints
                    </dt>
                    <ListItems items={project.scope.constraints} />
                  </div>
                )}
            </div>
          )}
        </SectionCard>

        {/* Governance */}
        <SectionCard
          title="Governance & Autonomy"
          empty="No governance settings"
        >
          {project.governance && (
            <div className="space-y-3">
              {project.governance.autonomy_level && (
                <div className="flex justify-between">
                  <span className="text-xs text-gray-500 uppercase">
                    Autonomy Level
                  </span>
                  <span className="text-sm font-medium text-gray-800">
                    {project.governance.autonomy_level}
                  </span>
                </div>
              )}
              {project.governance.review_strictness && (
                <div className="flex justify-between">
                  <span className="text-xs text-gray-500 uppercase">
                    Review Strictness
                  </span>
                  <span className="text-sm font-medium text-gray-800">
                    {project.governance.review_strictness}
                  </span>
                </div>
              )}
              {project.governance.decision_authority && (
                <div className="flex justify-between">
                  <span className="text-xs text-gray-500 uppercase">
                    Decision Authority
                  </span>
                  <span className="text-sm font-medium text-gray-800">
                    {project.governance.decision_authority}
                  </span>
                </div>
              )}
              {project.governance.human_in_the_loop &&
                project.governance.human_in_the_loop.length > 0 && (
                  <div>
                    <dt className="text-xs text-gray-500 uppercase mb-1">
                      Human-in-the-Loop Stages
                    </dt>
                    <ListItems items={project.governance.human_in_the_loop} />
                  </div>
                )}
            </div>
          )}
        </SectionCard>

        {/* Agent Config */}
        <SectionCard title="Agent Configuration" empty="No agent settings">
          {project.agent_config && (
            <div className="space-y-3">
              {project.agent_config.trust_level && (
                <div className="flex justify-between">
                  <span className="text-xs text-gray-500 uppercase">
                    Trust Level
                  </span>
                  <span className="text-sm font-medium text-gray-800">
                    {project.agent_config.trust_level}
                  </span>
                </div>
              )}
              {project.agent_config.max_parallel_agents !== undefined && (
                <div className="flex justify-between">
                  <span className="text-xs text-gray-500 uppercase">
                    Max Parallel Agents
                  </span>
                  <span className="text-sm font-medium text-gray-800">
                    {project.agent_config.max_parallel_agents}
                  </span>
                </div>
              )}
              {project.agent_config.allowed_agent_types &&
                project.agent_config.allowed_agent_types.length > 0 && (
                  <div>
                    <dt className="text-xs text-gray-500 uppercase mb-1">
                      Allowed Agent Types
                    </dt>
                    <div className="flex flex-wrap gap-1">
                      {project.agent_config.allowed_agent_types.map((type) => (
                        <span
                          key={type}
                          className="px-2 py-0.5 bg-blue-100 text-blue-800 text-xs rounded"
                        >
                          {type}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              {project.agent_config.runtime_permissions &&
                project.agent_config.runtime_permissions.length > 0 && (
                  <div>
                    <dt className="text-xs text-gray-500 uppercase mb-1">
                      Runtime Permissions
                    </dt>
                    <ListItems
                      items={project.agent_config.runtime_permissions}
                    />
                  </div>
                )}
            </div>
          )}
        </SectionCard>

        {/* Task Config */}
        <SectionCard title="Task Configuration" empty="No task settings">
          {project.task_config && (
            <div className="space-y-3">
              {project.task_config.allowed_task_types &&
                project.task_config.allowed_task_types.length > 0 && (
                  <div>
                    <dt className="text-xs text-gray-500 uppercase mb-1">
                      Allowed Task Types
                    </dt>
                    <div className="flex flex-wrap gap-1">
                      {project.task_config.allowed_task_types.map((type) => (
                        <span
                          key={type}
                          className="px-2 py-0.5 bg-purple-100 text-purple-800 text-xs rounded"
                        >
                          {type}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              {project.task_config.wip_limits &&
                Object.keys(project.task_config.wip_limits).length > 0 && (
                  <div>
                    <dt className="text-xs text-gray-500 uppercase mb-1">
                      WIP Limits
                    </dt>
                    <div className="space-y-1">
                      {Object.entries(project.task_config.wip_limits).map(
                        ([key, value]) => (
                          <div
                            key={key}
                            className="flex justify-between text-sm"
                          >
                            <span className="text-gray-600">{key}</span>
                            <span className="text-gray-800">{value}</span>
                          </div>
                        ),
                      )}
                    </div>
                  </div>
                )}
            </div>
          )}
        </SectionCard>

        {/* Members */}
        <SectionCard
          title="Team Members"
          empty={membersLoading ? "Loading..." : "No members found"}
        >
          {members.length > 0 && (
            <div className="space-y-2">
              {members.map((member) => (
                <div
                  key={member.id}
                  className="flex justify-between items-center py-1 border-b last:border-0"
                >
                  <span className="text-sm text-gray-800 font-mono">
                    {member.user_id.slice(0, 8)}...
                  </span>
                  <span
                    className={`px-2 py-0.5 rounded text-xs font-medium ${
                      member.role === "owner"
                        ? "bg-orange-100 text-orange-800"
                        : member.role === "admin"
                          ? "bg-red-100 text-red-800"
                          : member.role === "contributor"
                            ? "bg-blue-100 text-blue-800"
                            : member.role === "reviewer"
                              ? "bg-green-100 text-green-800"
                              : "bg-gray-100 text-gray-800"
                    }`}
                  >
                    {member.role}
                  </span>
                </div>
              ))}
            </div>
          )}
        </SectionCard>
      </div>

      {/* Footer Metadata */}
      <div className="mt-6 pt-4 border-t text-xs text-gray-400 flex gap-6">
        <span>Created: {new Date(project.created_at).toLocaleString()}</span>
        <span>Updated: {new Date(project.updated_at).toLocaleString()}</span>
        <span className="font-mono">ID: {project.id}</span>
      </div>
    </div>
  );
};

export const Route = createFileRoute("/_auth/projects/$projectId")({
  component: ProjectDetail,
});
