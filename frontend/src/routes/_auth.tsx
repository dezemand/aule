import { isAuthenticated } from "@/services/auth/api";
import { AuthProvider } from "@/services/auth/provider";
import { createFileRoute, redirect } from "@tanstack/react-router";

const AuthenticatedLayout = ({ children }: { children: React.ReactNode }) => {
  return <AuthProvider>{children}</AuthProvider>;
};

export const Route = createFileRoute("/_auth")({
  component: AuthenticatedLayout,
  beforeLoad: async ({ location }) => {
    if (!(await isAuthenticated())) {
      throw redirect({
        to: "/login",
        search: {
          redirect: location.href,
        },
      });
    }
  },
});
