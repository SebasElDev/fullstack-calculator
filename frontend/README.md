# Frontend — `@fullstack-calculator/frontend`

React 19 + Vite + TypeScript + Tailwind CSS v4 single-page app for the
calculator. It builds requests, renders responses and formats strings.

**No arithmetic happens here.** Every `+ − × ÷` and every unary operation is a
`POST /api/v1/calculate` round trip to the Go API. The only number conversions
in the source are `Number(input)` (parsing what the user typed) and
`String(result)` (rendering what the server returned).

## Scripts

| Command | What it does |
|---|---|
| `npm run dev` | Vite dev server on `:5173`, proxying `/api` and `/health` to `http://localhost:8080` |
| `npm run build` | `tsc -b && vite build` → `dist/` |
| `npm run preview` | Serve the production build locally |
| `npm run lint` | Biome lint + format check |
| `npm run format` | Biome format, writing changes |
| `npm run typecheck` | `tsc --noEmit -p tsconfig.app.json` |
| `npm run test` | Vitest, single pass |
| `npm run test:watch` | Vitest in watch mode |
| `npm run test:coverage` | Vitest with V8 coverage (85 % thresholds on `src/features` and `src/lib`) |

`VITE_API_BASE_URL` overrides the API base URL; it defaults to `/api/v1`
(same-origin, which is how the production container serves the SPA).

## Layout

```
src/
├── lib/api/{types,client}.ts   hand-written mirror of api/openapi.yaml + the typed client
├── lib/operations.ts           OPERATIONS: symbol, label, arity and kind per operation
├── lib/ids.ts                  session-unique ids for history rows
├── features/calculator/
│   ├── context/                ApiClientProvider (DI) and CalculatorProvider (state machine)
│   ├── state/                  reducer, formatting and keyboard bindings — string work only
│   └── components/             the compound <Calculator/> and its parts
└── test/                       jest-dom setup, renderWithProviders, createMockApiClient
```

## Contract discipline

`api/openapi.yaml` is the source of truth (docs/ARCHITECTURE.md §4). The types in
`src/lib/api/types.ts` mirror its schemas by hand, keeping the same names so the
two can be diffed by eye. If the contract grows past what is comfortable to
maintain by hand, generate them instead:

```sh
npx openapi-typescript ../api/openapi.yaml -o src/lib/api/schema.d.ts
```

### Cross-checking the operation registry

`src/lib/operations.ts` and `domain.Registry()` on the Go side must advertise the
same ten operations with the same symbols and arities. Two halves of that check
are automated, and one is manual:

1. **Frontend half (automated).** `src/lib/operations.test.ts` pins `OPERATIONS`
   against an independent copy of the §2.3 registry table and asserts that its
   keys are exactly the `OperationName` union. `npm run test` covers it.
2. **Backend half (automated, other track).** The Go tests assert
   `Operations()` matches `domain.Registry()` in order.
3. **Across the wire (manual).** Run the two against each other:

   ```sh
   # from the repository root, with the API running on :8080
   curl -s localhost:8080/api/v1/operations | python3 -m json.tool
   ```

   Compare each entry's `name`, `symbol` and `arity` with the corresponding row
   of `OPERATIONS` in `src/lib/operations.ts`. They must match exactly,
   including the glyphs (`−` U+2212, `×` U+00D7, `÷` U+00F7, `√` U+221A, `xʸ`,
   `x²`, `mod`, `±`, `%`). A mismatch means the two registries have drifted:
   fix whichever side disagrees with `api/openapi.yaml`.

`ApiClient.listOperations()` exists for exactly this discovery check
(docs/ARCHITECTURE.md §3.2). The UI does not call it — the keypad is built from
the local `OPERATIONS` table so it renders without a round trip — but the method
is part of the client's public surface so the endpoint can be exercised from a
console or a test.
