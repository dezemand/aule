import { queryKeys } from "@/lib/query";
import { useSubscription } from "@/services/websocket/use-subscription";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import type { FC } from "react";

const Projects: FC = () => {
  const { data } = useSubscription({
    queryKey: queryKeys.projects.list,
  });
  const projects = [{ id: "abc", name: "Hello!" }] as Array<{
    id: string;
    name: string;
  }>;

  return (
    <div className="p-2">
      {projects.length === 0 ? (
        <div>No projects found.</div>
      ) : (
        <ul>
          {projects.map((project) => (
            <li key={project.id}>
              <Link
                to={`/projects/$projectId`}
                params={{ projectId: project.id }}
              >
                {project.name}
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};

export const Route = createFileRoute("/_auth/projects/")({
  component: Projects,
});
