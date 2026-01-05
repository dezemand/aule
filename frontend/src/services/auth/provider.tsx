import type { FC, ReactNode } from "react";

type AuthProviderProps = {
  children: ReactNode;
};

export const AuthProvider: FC<AuthProviderProps> = ({ children }) => {
  return <>{children}</>;
};
