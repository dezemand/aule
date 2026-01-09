import { createContext } from "react";
import type { Claims } from "./store";

export interface AuthContextValue {
  /**
   * Sign out the user.
   */
  signOut: () => void;

  claims: Claims | null;
}

export const AuthContext = createContext<AuthContextValue | null>(null);
