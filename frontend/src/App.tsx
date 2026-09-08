import { ApiClientProvider, Calculator } from "@/features/calculator";
import { createApiClient, DEFAULT_API_BASE_URL } from "@/lib/api/client";

const apiClient = createApiClient({
  baseUrl: import.meta.env.VITE_API_BASE_URL ?? DEFAULT_API_BASE_URL,
});

export default function App() {
  return (
    <ApiClientProvider client={apiClient}>
      <main className="flex min-h-dvh flex-col items-center justify-center gap-5 px-3 py-8">
        <header className="w-full max-w-sm">
          <h1 className="text-base font-semibold tracking-tight text-slate-100">Calculator</h1>
        </header>
        <Calculator>
          <Calculator.Display />
          <Calculator.Keypad />
          <Calculator.History />
        </Calculator>
      </main>
    </ApiClientProvider>
  );
}
