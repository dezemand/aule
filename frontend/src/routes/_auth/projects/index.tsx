import { queryKeys } from "@/lib/query";
import { useSubscription } from "@/services/subscriptions/use-subscription";
import { createFileRoute, Link } from "@tanstack/react-router";
import type { FC } from "react";
import type { Project, ProjectsListResponse } from "@/model/ws";

const StatusBadge: FC<{ status: Project["status"] }> = ({ status }) => {
  const colors = {
    active: "bg-green-100 text-green-800",
    paused: "bg-yellow-100 text-yellow-800",
    archived: "bg-gray-100 text-gray-800",
  };

  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${colors[status]}`}>
      {status}
    </span>
  );
};

const Projects: FC = () => {
  const { data, isLoading } = useSubscription<ProjectsListResponse>({
    queryKey: queryKeys.projects.list,
    topic: "projects.list",
  });

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Projects</h1>
        <Link
          to="/"
          className="text-sm text-gray-500 hover:text-gray-700"
        >
          Back to Home
        </Link>
      </div>

      {isLoading ? (
        <div className="text-center py-12">
          <div className="animate-pulse text-gray-500">Loading projects...</div>
        </div>
      ) : !data?.payload.projects || data?.payload.projects.length === 0 ? (
        <div className="text-center py-12 bg-gray-50 rounded-lg">
          <p className="text-gray-500 mb-4">No projects found.</p>
          <p className="text-sm text-gray-400">
            Projects will appear here once created.
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {data.payload.projects.map((project) => (
            <Link
              key={project.id}
              to={`/projects/$projectId`}
              params={{ projectId: project.id }}
              className="block p-4 bg-white border rounded-lg hover:shadow-md transition-shadow"
            >
              <div className="flex justify-between items-start mb-2">
                <div>
                  <h2 className="text-lg font-semibold text-gray-900">
                    {project.name}
                  </h2>
                  <span className="text-xs text-gray-400 font-mono">
                    {project.key}
                  </span>
                </div>
                <StatusBadge status={project.status} />
              </div>
              
              {project.description && (
                <p className="text-gray-600 text-sm mb-3">
                  {project.description}
                </p>
              )}
              
              {project.purpose?.goal && (
                <p className="text-gray-500 text-sm italic">
                  Goal: {project.purpose.goal}
                </p>
              )}
              
              <div className="flex gap-4 mt-3 text-xs text-gray-400">
                <span>Created: {new Date(project.created_at).toLocaleDateString()}</span>
                <span>Updated: {new Date(project.updated_at).toLocaleDateString()}</span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
};

export const Route = createFileRoute("/_auth/projects/")({
  component: Projects,
});
