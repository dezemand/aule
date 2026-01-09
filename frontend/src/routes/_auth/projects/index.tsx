import { queryKeys } from "@/lib/query";
import {
  useSubscribe,
  useSubscription,
} from "@/services/subscriptions/use-subscription";
import { createFileRoute, Link } from "@tanstack/react-router";
import type { FC } from "react";

const Projects: FC = () => {
  const { data, status } = useSubscription({
    topic: "projects.list",
    query: null,
  });

  return (
    <div className="p-2">
      <p>Subscription: {status}</p>
      <pre>{JSON.stringify(data, null, 2)}</pre>
      <p>
        <Link to="/">Go back</Link>
      </p>

      {/*{projects.length === 0 ? (
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
      )}*/}
    </div>
  );
};

export const Route = createFileRoute("/_auth/projects/")({
  component: Projects,
});
