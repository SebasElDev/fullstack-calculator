import { createContext, type ReactNode, useContext } from "react";
import type { ApiClient } from "@/lib/api/client";

const ApiClientContext = createContext<ApiClient | null>(null);

export interface ApiClientProviderProps {
  /** Production injects `createApiClient(...)`; tests inject a mock. */
  client: ApiClient;
  children: ReactNode;
}

/** Dependency injection for the typed API client. */
export function ApiClientProvider({ client, children }: ApiClientProviderProps) {
  return <ApiClientContext value={client}>{children}</ApiClientContext>;
}

export function useApiClient(): ApiClient {
  const client = useContext(ApiClientContext);
  if (client === null) {
    throw new Error("useApiClient must be used inside <ApiClientProvider>");
  }
  return client;
}
