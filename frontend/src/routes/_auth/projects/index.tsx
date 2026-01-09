import { queryKeys } from "@/lib/query";
import {
  useSubscribe,
  useSubscription,
} from "@/services/subscriptions/use-subscription";
import { createFileRoute, Link } from "@tanstack/react-router";
import type { FC } from "react";

interface ProjectsResult {
  projects: {
    id: string;
    name: string;
    description: string;
  }[];
}

const Projects: FC = () => {
  const { data, isLoading } = useSubscription<ProjectsResult>({
    queryKey: queryKeys.projects.list,
    topic: "projects.list",
  });

  return (
    <div className="p-2">
      <p>
        <Link to="/">Go back</Link>
      </p>

      {isLoading ? (
        <div>Loading projects...</div>
      ) : !data?.payload.projects || data?.payload.projects.length === 0 ? (
        <div>No projects found.</div>
      ) : (
        <div>
          <h2>Projects</h2>
          <ul>
            {data?.payload.projects.map((project) => (
              <li key={project.id}>
                <Link
                  to={`/projects/$projectId`}
                  params={{ projectId: project.id }}
                >
                  {project.name}
                </Link>
                <br />
                {project.description}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
};

export const Route = createFileRoute("/_auth/projects/")({
  component: Projects,
});
