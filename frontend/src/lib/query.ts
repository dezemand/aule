import { QueryClient } from "@tanstack/react-query";

const defaultQueryOptions = {
  queries: {
    // Stale time: 5 minutes for list queries
    staleTime: 5 * 60 * 1000,

    // Cache time: 30 minutes (how long inactive data stays in cache)
    gcTime: 30 * 60 * 1000,

    // Refetch on window focus to keep data fresh
    refetchOnWindowFocus: true,

    // Refetch on reconnect after network errors
    refetchOnReconnect: true,

    // Don't refetch on mount if data is fresh
    refetchOnMount: false,

    // Retry failed requests with exponential backoff
    retry: (failureCount: number, error: unknown) => {
      // Don't retry on 4xx errors (client errors)
      if (error && typeof error === "object" && "code" in error) {
        const code = String((error as { code?: string | number }).code);
        if (code.startsWith("4")) {
          return false;
        }
      }

      // Retry up to 3 times for other errors (5xx, network errors)
      return failureCount < 3;
    },

    // Exponential backoff delay
    retryDelay: (attemptIndex: number) =>
      Math.min(1000 * 2 ** attemptIndex, 30000),
  },

  mutations: {
    // Retry mutations once on failure
    retry: 1,

    // Don't retry on 4xx errors
    retryDelay: 1000,
  },
};

export const queryClient = new QueryClient({
  defaultOptions: defaultQueryOptions,
});

export const queryKeys = {
  auth: {
    start: ["auth", "start"],
    providers: ["auth", "providers"],
  },
  projects: {
    list: ["projects", "list"],
    detail: (id: string) => ["projects", "detail", id] as const,
    members: (id: string) => ["projects", id, "members"] as const,
    repos: (id: string) => ["projects", id, "repos"] as const,
  },
} as const;
