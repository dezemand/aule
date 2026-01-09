import { useAuthStore } from "@/services/auth/store";
import { useAuth } from "@/services/auth/use-auth";
import { createFileRoute, Link } from "@tanstack/react-router";
import type { FC } from "react";

const Index: FC = () => {
  const { signOut } = useAuth();
  const { claims } = useAuthStore();

  return (
    <div className="p-2">
      <h3>Welcome Home, {claims?.id}!</h3>
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
