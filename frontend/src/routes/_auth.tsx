import { isAuthenticated } from "@/services/auth/api";
import { AuthProvider } from "@/services/auth/provider";
import {
  createFileRoute,
  Outlet,
  redirect,
  useRouter,
} from "@tanstack/react-router";
import { useCallback } from "react";

function AuthenticatedLayout() {
  const router = useRouter();

  const handleAuthFailure = useCallback(() => {
    router.navigate({
      to: "/login",
      search: {
        redirect: window.location.pathname,
      },
    });
  }, [router]);

  return (
    <AuthProvider onAuthFailure={handleAuthFailure}>
      <Outlet />
    </AuthProvider>
  );
}

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
