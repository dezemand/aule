import { useAuth } from "@/services/auth/use-auth";
import { createFileRoute, Link } from "@tanstack/react-router";
import type { FC } from "react";

const Index: FC = () => {
  const { signOut } = useAuth();

  return (
    <div className="p-2">
      <h3>Welcome Home!</h3>
      <p>
        <Link to="/projects">Projects</Link>
      </p>
      <button type="button" onClick={signOut}>
        Logout
      </button>
    </div>
  );
};

export const Route = createFileRoute("/_auth/")({
  component: Index,
});
