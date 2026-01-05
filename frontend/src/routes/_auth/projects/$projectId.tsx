import { createFileRoute } from "@tanstack/react-router"
import type { FC } from "react";

const Project: FC = () => {
  const { projectId } = Route.useParams();

  return (
    <div>
      <h2>Project {projectId}</h2>
    </div>
  );
};

export const Route = createFileRoute("/_auth/projects/$projectId")({
  component: Project,
});
