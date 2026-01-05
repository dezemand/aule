import { queryOptions, useQuery } from "@tanstack/react-query";
import { getAuleAuthAPI } from "./api.gen";
import { queryKeys } from "@/lib/query";
import { storeAuthToken } from "@/lib/client";

export const authProviders = queryOptions({
  queryKey: queryKeys.auth.providers,
  queryFn: () => getAuleAuthAPI().getProviders(),
  staleTime: Infinity, // It will not change unless the backend restarts.
});

export function useAuthProviders() {
  return useQuery(authProviders);
}

export async function authFromCallback(
  provider: string,
  state: string,
  code: string,
) {
  const response = await getAuleAuthAPI().callbackOAuth(provider, {
    state,
    code,
  });
  storeAuthToken(response.token);
}

export async function isAuthenticated(): Promise<boolean> {
  return true;
}
