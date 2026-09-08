/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Base URL of the Calculator API; defaults to `/api/v1`. See `.env.example`. */
  readonly VITE_API_BASE_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
