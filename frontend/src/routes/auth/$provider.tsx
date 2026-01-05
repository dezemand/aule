import {
  createFileRoute,
  Link,
  Navigate,
  SearchParamError,
  useParams,
  type ErrorComponentProps,
} from "@tanstack/react-router";
import { useEffect, type FC } from "react";
import { z } from "zod";
import { zodValidator } from "@tanstack/zod-adapter";
import { getAuleAuthAPI } from "@/services/auth/api.gen";
import { AxiosError } from "axios";
import { authFromCallback } from "@/services/auth/api";

class OAuthError extends Error {
  constructor(
    public error: string,
    public error_description?: string,
  ) {
    super(`OAuth Error: ${error} - ${error_description}`);
    this.name = "OAuthError";
  }
}

const authReturnSuccessSearchSchema = z.object({
  state: z.string(),
  code: z.string(),
});

const authReturnErrorSearchSchema = z.object({
  error: z.string(),
  error_description: z.string().optional(),
});

const authReturnSearchSchema = z.union([
  authReturnSuccessSearchSchema,
  authReturnErrorSearchSchema,
]);

const Return: FC = () => {
  return <Navigate to="/" replace />;
};

const ReturnError: FC<ErrorComponentProps> = ({ error }) => {
  if (error instanceof SearchParamError) {
    return (
      <div>
        Invalid URL, <Link to="/login">Go back</Link>
      </div>
    );
  }

  if (error instanceof AxiosError) {
    return (
      <div>
        API Error: {error.response?.status}, <Link to="/login">Go back</Link>
      </div>
    );
  }

  if (error instanceof OAuthError) {
    return (
      <div>
        OAuth Error: {error.error} - {error.error_description},{" "}
        <Link to="/login">Go back</Link>
      </div>
    );
  }

  return (
    <div>
      Error: {error.message}, <Link to="/login">Go back</Link>
    </div>
  );
};

export const Route = createFileRoute("/auth/$provider")({
  component: Return,
  errorComponent: ReturnError,
  validateSearch: zodValidator(authReturnSearchSchema),
  loaderDeps: ({ search }) => ({ search }),
  loader: ({ params: { provider }, deps: { search } }) => {
    if ("error" in search) {
      throw new OAuthError(search.error, search.error_description);
    }
    return authFromCallback(provider, search.state, search.code);
  },
});
