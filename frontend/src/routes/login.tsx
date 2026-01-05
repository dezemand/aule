import { createFileRoute, redirect } from "@tanstack/react-router";
import { authProviders, isAuthenticated } from "@/services/auth/api";
import type { FC } from "react";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { queryClient, queryKeys } from "@/lib/query";
import { getAuleAuthAPI } from "@/services/auth/api.gen";

export const Login: FC = () => {
  const client = useQueryClient();
  const { data } = useSuspenseQuery(authProviders);

  const sendToLogin = async (provider: string) => {
    const { authUrl } = await client.fetchQuery({
      queryKey: queryKeys.auth.start,
      queryFn: () => getAuleAuthAPI().startOAuth(provider),
    });
    window.location.href = authUrl;
  };

  return (
    <>
      <h2>Login</h2>
      <ul>
        {data.providers?.map((provider) => (
          <li key={provider.id}>
            <button type="button" onClick={() => sendToLogin(provider.id)}>
              Login with {provider.name}
            </button>
          </li>
        ))}
      </ul>
    </>
  );
};

export const Route = createFileRoute("/login")({
  beforeLoad: async () => {
    if (await isAuthenticated()) {
      throw redirect({ to: "/" });
    }
  },
  loader: () => queryClient.ensureQueryData(authProviders),
  component: Login,
});
