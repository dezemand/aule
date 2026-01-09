import { jsonCodec } from "@/lib/utils";
import { z } from "zod";
import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";

/** The JWT claims part schema */
export const claimsSchema = jsonCodec(
  z.object({
    id: z.uuid(),
    role: z.enum(["user"]),
    exp: z.number(),
  }),
);
export type Claims = z.infer<typeof claimsSchema>;

/** The localStorage token store schema */
const storeTokenSchema = jsonCodec(
  z.object({
    token: z.string(),
  }),
);

type AuthState = {
  token: string | null;
  claims: Claims | null;

  setToken: (token: string | null) => void;
};

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      claims: null,
      setToken: (token) =>
        set({
          token,
          claims: token ? getTokenClaims(token) : null,
        }),
    }),
    {
      name: "aule/auth",
      version: 1,
      storage: createJSONStorage(() => localStorage),
      partialize: (s) => ({ token: s.token }) as const,
      onRehydrateStorage: () => (state, error) => {
        if (error) {
          state?.setToken(null);
          return;
        }

        try {
          const raw = {
            token: state?.token,
          };

          const parsed = storeTokenSchema.parse(raw);
          state?.setToken(parsed.token);
        } catch {
          // Invalid or tampered localStorage → reset
          state?.setToken(null);
        } finally {
        }
      },
    },
  ),
);

export const auth = {
  getToken: () => useAuthStore.getState().token,
  setToken: (t: string | null) => useAuthStore.getState().setToken(t),
  clearToken: () => useAuthStore.getState().setToken(null),
};

function getTokenClaims(token: string): Claims {
  return claimsSchema.decode(atob(token.split(".")[1]));
}

/**
 * Check if a JWT token is expired or will expire within the buffer time.
 */
export function isTokenValid(token: string, bufferSeconds = 60): boolean {
  try {
    const exp = getTokenClaims(token).exp * 1000;
    const now = Date.now();
    const buffer = bufferSeconds * 1000;
    return exp - now > buffer;
  } catch {
    // If we can't parse the token, assume it's invalid
    return false;
  }
}
