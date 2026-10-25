# Teaching Support Web Application

Teaching Support provides courses, assignments and reusable templates, file-backed
student submissions, grading and feedback, assignment statistics, Python execution,
and optional AI grading. The Go module retains the historical repository name
`Tech-Support-CN101`.

Start with the [development checklist](docs/development-checklist.md),
[architecture map](docs/architecture.md), [domain rules](docs/domain-rules.md),
[quality baseline](docs/quality-baseline.md), and [working rules](AGENTS.md).

## What is verified

The 2026-10-24 Windows baseline passed frontend typecheck/build, Go compilation,
Go vet, and API/worker binary builds. F4 frontend lint passes with 0 errors and
74 existing warnings (down from 39 errors and 79 warnings);
frontend tests are not configured. R1 added and passed PostgreSQL-backed HTTP tests
for nested submission authorization; see the baseline for the required test URL.
R2 added storage isolation tests using local files and a fake S3 HTTP endpoint,
including SDK multipart uploads; they require no external services or credentials.
R3 verifies JWT configuration and expiry; R4 verifies human-grade protection with
disposable PostgreSQL and a fake AI endpoint. Both database suites use the explicit
test URL documented in the baseline and skip without it.
A frontend development-server HTTP smoke check also passed. These results do not verify
database, university authentication, storage, execution, or grading integrations.

The full Compose stack is **not yet a verified one-command setup**. Resolve the
configuration blockers below before using it for integration testing. Docker was
not available on PATH during verification.

## Repository layout

| Path | Purpose |
| --- | --- |
| `frontend/src/app/` | Next.js App Router pages |
| `frontend/src/components/` | Shared UI, assignment/template dialogs, editors |
| `frontend/src/lib/auth-context.tsx` | Client authentication state and API login |
| `backend/cmd/server/` | Fiber v3 HTTP API on port 8080 |
| `backend/cmd/worker/` | Asynq background grading worker |
| `backend/internal/` | Handlers, services, repositories, models, storage, AI adapters |
| `docker/docker-compose.yml` | Primary full-stack integration configuration |
| `backend/docker-compose.yml` | Older partial stack; not the setup documented here |
| `docs/` | Checklist and verified quality results |

The frontend uses React, TypeScript, Tailwind/shadcn components, TipTap, and Monaco.
The backend uses Fiber, GORM/PostgreSQL, Redis/Asynq, and S3/MinIO/local storage.
Judge0 executes Python; AI adapters support OpenAI, Anthropic, Google, and custom
OpenAI-compatible endpoints. Grafana/Loki/Promtail provide log collection/viewing;
metrics and deployed monitoring behavior have not been verified.

## Local compile and frontend development

Commands use PowerShell 7 and start at the repository root. Use Git, Go 1.25.3,
Node.js 24.13.0, and npm 11.9.0 to reproduce the measured host baseline. These are
observed working versions, not a newly enforced version policy. The frontend
Dockerfile currently uses Node 25; container compatibility is unverified.

Keep the existing npm lockfile and Go module files. Install only the existing
application dependencies; no AI tooling is needed.

### Frontend

```powershell
Set-Location frontend
$env:NEXT_PUBLIC_API_URL = 'http://127.0.0.1:8080/api'
$env:NEXT_TELEMETRY_DISABLED = '1'
npm ci --no-audit --no-fund
npm exec -- tsc --noEmit --incremental false
npm run lint
npm run build
```

Lint currently exits with code 0 and reports 74 warnings; record it independently
of the typecheck/build results. Do not add suppressions to hide failures. The
install command skips
npm's vulnerability audit; it is not a security-scan result.

To preview the frontend:

```powershell
npm run dev -- --hostname 127.0.0.1 --port 3300
```

Open [the login page](http://127.0.0.1:3300/login). Visiting the root redirects to
`/login`. Stop the server with Ctrl+C. This previews the frontend without starting
the backend; login and course operations still require the API and its services.
See [frontend/README.md](frontend/README.md) for frontend-specific details.

### Backend compilation

From a new terminal at the repository root:

```powershell
Set-Location backend
$env:GOTOOLCHAIN = 'local'
go test -mod=readonly ./...
go vet -mod=readonly ./...
go build -mod=readonly ./...
```

These commands do not start the API or worker. Submission integration tests skip
unless `TEACHING_SUPPORT_TEST_DATABASE_URL` is supplied. See the baseline's R1
section for safe test setup and the separately verified API/worker binary builds.

## Runtime configuration

No credentials are needed for the compilation checks. Runtime integration requires
dedicated development services and credentials. Do not point development processes
at an existing or production database: **both API and worker startup run GORM
AutoMigrate**. MinIO initialization also attempts to create its bucket; local
storage initialization creates directories.

`backend/internal/config/config.go` loads `.env` from the process working directory.
For native runs from `backend/`, that means `backend/.env`. Existing process
environment values take precedence. After loading dotenv, the API validates
`JWT_SECRET` before database initialization: it must contain at least 32 bytes and
must not be blank. Signing and verification use that configured value.
Sessions expire eight hours after login; expired or legacy tokens without expiry
receive HTTP 401. Log out and sign in again; there is no refresh-token flow or
automatic browser redirect on expiry. The worker does not issue or verify JWTs.

| Variables | When needed / actual behavior |
| --- | --- |
| `DATABASE_URL` | API and worker startup; PostgreSQL DSN for a disposable database |
| `JWT_SECRET` | API signing/verification; generated secret of at least 32 bytes, supplied through the process environment or dotenv |
| `REDIS_ADDRESS`, `REDIS_PASSWORD` | Queue/worker connection; address is `host:port`, password must match Redis |
| `STORAGE_TYPE` | Choose `local`, `minio`, or `s3` explicitly; unset defaults to `s3` |
| `LOCAL_STORAGE_PATH` | Local storage directory; default `./uploads`; API/worker must share access |
| `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET` | Required for MinIO; endpoint must include its scheme, for example `http://127.0.0.1:9000` for native runs |
| `MINIO_USE_SSL` | Parsed as a boolean, but currently does not change the endpoint; not an effective TLS switch |
| `S3_BUCKET`, `AWS_REGION`, AWS credential configuration | S3 storage; AWS SDK default configuration/credential chain is used |
| `TU_API` | University application key required for actual login/profile requests |
| `JUDGE0_URL`, `JUDGE0_API_KEY` | Python execution; code requires both and sends the key as `X-API-Key` |
| `AI_SECRET_KEY` | AI credential encryption/decryption; raw AES key of 16, 24, or 32 bytes, shared by API/worker; preserve it for saved credentials |
| `ENV`, `SERVICE_NAME` | Logging: `ENV=production` selects JSON logs; service name labels them |

AI provider keys are entered through the application's credential workflow and
stored encrypted in PostgreSQL. An `OPENAI_API_KEY` environment variable is not an
application setup requirement in the current source. There is no documented local
authentication bypass or seeded account; actual login calls university services.

For a native development session, first provision an empty PostgreSQL database
and a development Redis instance independently. In a terminal whose working
directory is `backend/`, set `DATABASE_URL`, `REDIS_ADDRESS`, `REDIS_PASSWORD`,
and a generated `JWT_SECRET` in the environment. Set `STORAGE_TYPE=local` and
`LOCAL_STORAGE_PATH` to a dedicated external development directory to avoid
committing uploaded files. Add integration-specific variables only when needed.
After verifying those targets, the entry points are:

```powershell
# API: only against the dedicated disposable database
go run -mod=readonly ./cmd/server
```

For the worker, use a separate terminal with the same
database/queue/storage/AI key settings:

```powershell
go run -mod=readonly ./cmd/worker
```

These runtime commands were not executed during F1/F2. The API listens on all
interfaces at port 8080; use a controlled local environment. Start the worker only
when testing grading jobs, as queued tasks can call paid AI providers and write
scores/comments.

## Compose integration configuration

Use `docker/docker-compose.yml` for future full-stack integration work.
`backend/docker-compose.yml` omits the frontend, MinIO, and Judge0, differs in Redis
version and persistence, and conflicts with frontend port 3000 through Grafana.
Do not start both stacks together.

The primary file defines **13 services**: API, worker, frontend, PostgreSQL,
application Redis, MinIO, Judge0 server/workers/database/Redis, Loki, Promtail, and
Grafana. It publishes these host ports:

| Service | Host port(s) |
| --- | --- |
| Frontend | 3000 |
| API | 8080 |
| PostgreSQL | 5432 |
| MinIO API / console | 9000 / 9001 |
| Grafana | 3001 |

Other services have no host port mappings. Both named networks are shared bridge
networks; the name `private` does not itself provide an isolated execution boundary.
Judge0 containers are privileged. Use a disposable Linux Docker environment for
integration validation; the log shipper mounts Linux Docker container paths.
Fixed container names and ports mean a different Compose project name alone does
not guarantee separation from existing services.

Two environment files have different roles:

- `docker/.env` supplies Compose interpolation, using `docker/.env.example` as a
  list of variables for databases, Redis, MinIO, Grafana, and frontend build URL.
- `backend/.env` supplies API/worker runtime variables via Compose `env_file`.
  Explicit Compose `environment` entries override the corresponding values.
  API JWT configuration is validated after environment loading and before DB startup.

Both local `.env` files are ignored. Do not use the tracked `backend/.env copy`
as a secure template; it contains fixed credential defaults. Fill values using
development credentials and the runtime table above. No `.dockerignore` exists
yet, so a local environment file may enter a Docker build context (O2).

### Blockers before a full-stack start

1. **O1: MinIO endpoint.** API and worker explicitly override `MINIO_ENDPOINT` with
   `minio:9000`; the storage adapter uses it directly as a URL. A
   `backend/.env` value cannot correct that override. The integration endpoint
   needs its scheme; the source/Compose repair is separate work.
2. **O1: Judge0 configuration.** The API requires `JUDGE0_API_KEY`, which the
   existing examples omit. Verify engine authentication against the actual
   `X-API-Key` header instead of assuming any nonempty key secures Judge0.
   Passwords in mounted `docker/judge0.conf` are static, not interpolated from
   `docker/.env`; keep its DB/Redis settings consistent with those containers.
3. **O3: Readiness and persistence.** The primary stack has no readiness
   healthchecks, application Redis has no durable volume, and both Go processes
   migrate the same schema at startup. Integration startup/recovery needs validation.

After creating dedicated development configuration and resolving these blockers,
run the following **from the repository root**, first validating without starting
services:

```powershell
docker compose --env-file docker/.env -f docker/docker-compose.yml config --quiet
```

Then, only in the disposable integration environment:

```powershell
docker compose --env-file docker/.env -f docker/docker-compose.yml up -d --build
docker compose --env-file docker/.env -f docker/docker-compose.yml ps
docker compose --env-file docker/.env -f docker/docker-compose.yml logs --tail 100 api worker
```

A running container list does not prove the application integrations work. Verify
login, upload/download, submission/grade persistence, queue completion, and Judge0
separately after setup repairs. Do not publish logs containing credentials or
student content. Stop the disposable stack with `docker compose --env-file
docker/.env -f docker/docker-compose.yml down`; volume deletion is a separate,
destructive operation.

Reverse-proxy, TLS, Coolify, backups, and production rollout require their own
verified deployment procedure. No reverse proxy is included in this Compose file.
Do not use an image pull or production deployment as a local setup check.

## Next work

Follow the checklist for setup repairs (O1-O3), authorization and grading fixes,
tests, and minimal CI. AI tooling, indexes, graphs, memories, and source exports
must remain physically outside the repository, including ignored directories.
