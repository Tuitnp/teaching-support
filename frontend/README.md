# Teaching Support frontend

This is the Next.js App Router frontend for Teaching Support. Use npm and the
existing `package-lock.json`. The source is under `src/app/`, with shared UI in
`src/components/` and authentication in `src/lib/auth-context.tsx`.

For runtime services and environment details, see the [root setup guide](../README.md).
For known failures and measured versions, see the
[quality baseline](../docs/quality-baseline.md).

## Local development

The verified host environment is Windows, PowerShell 7, Node.js 24.13.0, and
npm 11.9.0. The lockfile resolves Next.js 16.2.6, React 19.2.0, and TypeScript
5.9.3. These results do not yet verify the Dockerfile's Node 25 environment.

From `frontend/`:

```powershell
$env:NEXT_PUBLIC_API_URL = 'http://127.0.0.1:8080/api'
$env:NEXT_TELEMETRY_DISABLED = '1'
npm ci --no-audit --no-fund
npm run dev -- --hostname 127.0.0.1 --port 3300
```

Open [the login page](http://127.0.0.1:3300/login). The root redirects to `/login`.
Stop with Ctrl+C. The F2 HTTP smoke check verified startup, the redirect, and a
200 response from `/login`; it did not exercise browser interaction or login.

`NEXT_PUBLIC_API_URL` is the **browser-reachable API base URL including `/api`**.
For native local development this is usually `http://127.0.0.1:8080/api`.
Do not use Docker service names such as `api:8080` for a browser running on the host.

The root instructions explain how to run the API against a dedicated development
database. The frontend can start without it, but authentication and course data
require the backend and university authentication. No mock login is configured.
Client authentication persists in browser local storage; use a separate browser
profile for development accounts.

For a persistent local value, `frontend/.env.local` may contain the API URL above.
This file is ignored; keep actual secrets out of `NEXT_PUBLIC_*` variables.
The commands above use process-local settings and create no environment file.

## Quality checks

```powershell
npm exec -- tsc --noEmit --incremental false
npm run lint
npm run test:datetime
npm run build
```

Typecheck and production build passed in F1 and F4. F4 cleared the 39 lint errors;
lint now passes with 74 existing warnings, down from 79. Run it separately and avoid
adding new warnings or suppressions. R8 adds `npm run test:datetime`, using Node 24
built-in TypeScript support and `node:test` without another dependency. It checks Bangkok display/parsing, unchanged
edit precision, and DST-safe time changes under three host timezones. There is no
general `test` script or component test suite: `npm test`
currently exits with "Missing script".

The app loads JetBrains Mono through `next/font/google` in `src/app/layout.tsx`;
the initial development/build compilation may need network access for the font.
The UI uses TipTap for rich text and Monaco for code, alongside shared
Tailwind/shadcn components.

## Production build behavior

Set `NEXT_PUBLIC_API_URL` before `npm run build`. Its public value is embedded in
browser assets; changing only a running container's environment does not update
those assets. The Dockerfile receives it as a build argument from the primary
Compose file. Rebuild when changing the browser API target.

`npm run start` serves an existing production build. It does not start the Go API
or its dependencies. Full-stack Compose startup has known configuration blockers;
follow the root guide rather than the older partial backend Compose file.

Keep generated `.next/`, `next-env.d.ts`, dependencies, and local environment
files uncommitted. Keep AI-specific tooling/state outside the repository.
