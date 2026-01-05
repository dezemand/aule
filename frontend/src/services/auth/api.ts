import { queryOptions, useQuery } from "@tanstack/react-query";
import { getAuleAuthAPI } from "./api.gen";
import { queryKeys } from "@/lib/query";
import { clearAuthToken, getAuthToken, storeAuthToken } from "@/lib/client";

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

export async function logout() {
  await getAuleAuthAPI().revokeRefreshToken();
  clearAuthToken();
}

/**
 * Check if a JWT token is expired or will expire within the buffer time.
 */
function isTokenValid(token: string, bufferSeconds = 60): boolean {
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    const exp = payload.exp * 1000; // Convert to milliseconds
    const now = Date.now();
    const buffer = bufferSeconds * 1000;
    return exp - now > buffer;
  } catch {
    // If we can't parse the token, assume it's invalid
    return false;
  }
}

/**
 * Attempt to refresh the JWT token using the refresh token cookie.
 * Returns true if successful, false otherwise.
 */
async function tryRefreshToken(): Promise<boolean> {
  try {
    const response = await getAuleAuthAPI().getJwt();
    if (response.token) {
      storeAuthToken(response.token);
      return true;
    }
    return false;
  } catch (error) {
    console.error("Token refresh failed:", error);
    return false;
  }
}

/**
 * Check if the user is authenticated.
 * First checks for a valid local token, then attempts refresh if needed.
 */
export async function isAuthenticated(): Promise<boolean> {
  // Check for existing token
  const token = getAuthToken();

  if (token && isTokenValid(token)) {
    // Token exists and is valid
    return true;
  }

  // No valid token, try to refresh using refresh token cookie
  return tryRefreshToken();
}

/**
 * Refresh the JWT token using the refresh token cookie.
 * Returns the new token or null if refresh failed.
 */

async function refreshToken(): Promise<string | null> {
  try {
    const response = await getAuleAuthAPI().getJwt();
    if (response.token) {
      storeAuthToken(response.token);
      return response.token;
    }
    return null;
  } catch (error) {
    console.error("Token refresh failed:", error);
    return null;
  }
}
/**
 * Check if a JWT token is expiring within the next minute.
 */
function isTokenExpiringSoon(token: string): boolean {
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    const exp = payload.exp * 1000; // Convert to milliseconds
    const now = Date.now();
    const oneMinute = 60 * 1000;
    return exp - now < oneMinute;
  } catch {
    // If we can't parse the token, assume it's expired
    return true;
  }
}
/**
 * Get a valid auth token, refreshing if necessary.
 */
export async function getValidToken(): Promise<string | null> {
  // First try to use existing token
  const existingToken = getAuthToken();
  if (existingToken) {
    // Check if token is still valid (not expired)
    // JWT tokens have 15 min expiry, so we refresh if < 1 min remaining
    if (!isTokenExpiringSoon(existingToken)) {
      return existingToken;
    }
  }

  // Token expired or expiring soon, try to refresh
  return refreshToken();
}
