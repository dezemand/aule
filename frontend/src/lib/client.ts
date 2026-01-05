import axios, {
  AxiosError,
  AxiosHeaders,
  type AxiosRequestConfig,
  type InternalAxiosRequestConfig,
} from "axios";
import { jsonCodec } from "./utils";
import z from "zod";

const API_BASE_URL = "/api";
export type ErrorType<Error> = AxiosError<Error>;
export type BodyType<BodyData> = BodyData;
const storeTokenSchema = jsonCodec(
  z.object({
    token: z.string(),
  }),
);

// const isDevelopment = import.meta.env.DEV;
// const API_BASE_URL = isDevelopment
//   ? "" // Use relative paths in development to go through Vite proxy
//   : import.meta.env.VITE_API_BASE_URL || "";

export const AXIOS_INSTANCE = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    "Content-Type": "application/json",
  },
  timeout: 30000,
});

function isAuthRoute(config: InternalAxiosRequestConfig) {
  return /^\/auth(\/.+)?$/.test(config.url);
}

AXIOS_INSTANCE.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    if (isAuthRoute(config)) {
      return config;
    }

    const authToken = getAuthToken();
    if (authToken) {
      config.headers ??= new AxiosHeaders();
      config.headers.set("Authorization", `Bearer ${authToken}`);
    }

    return config;
  },
);

export function getClient<T>(
  config: AxiosRequestConfig,
  options?: AxiosRequestConfig,
): Promise<T> {
  const promise = AXIOS_INSTANCE({
    ...config,
    ...options,
  }).then(({ data }) => data);

  return promise;
}

export function storeAuthToken(token: string) {
  const encoded = storeTokenSchema.encode({ token });
  window.localStorage.setItem("auth", encoded);
}

export function getAuthToken(): string | null {
  const stored = window.localStorage.getItem("auth");
  if (!stored) {
    return null;
  }
  const obj = storeTokenSchema.decode(stored);
  return obj.token;
}
