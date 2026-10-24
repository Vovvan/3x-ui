# 3x-ui — Architecture & Code Map

> Navigation map for contributors and AI coding agents (referenced from `CLAUDE.md`).
> Goal: jump to the right file in one hop instead of grepping the whole tree.
> Tracks the `main` branch — paths reflect the latest changes, so verify against the live
> tree rather than a pinned release (Go module `github.com/mhsanaei/3x-ui/v3`).
>
> **How to use this file:** read "Mental model" + "Request lifecycle" first, then
> use the **Symptom → File index** to locate work. Respect the **Layering rules**
> when adding code. Verify with the commands in **Build / Test / Lint**.

---

## 1. Mental model (the 30-second version)

3x-ui is a **web control panel for [Xray-core](https://github.com/XTLS/Xray-core)**. The Go
backend is the source of truth: it stores inbounds/clients/settings in a DB, renders an
Xray JSON config from that state, supervises the Xray child process, and exposes a REST +
WebSocket API. A React SPA (built by Vite, embedded into the Go binary) is the UI.
`internal/sub` is a pure **share-link builder** used by the panel's export and QR paths.

The panel supervises **managed child processes**: Xray-core itself and — when MTProto
inbounds exist — a dedicated child proxy binary:

- **`mtg-multi` for MTProto inbounds** (`github.com/mhsanaei/mtg-multi`, a multi-secret fork
  built from source; `internal/mtproto/`): One process per inbound serves every attached
  client's FakeTLS secret through the fork's `[secrets]` section, plus optional per-client
  sponsored-channel ad-tags via `[secret-ad-tags]`. A client or ad-tag edit is hot-applied via
  the fork's management API (`PUT /secrets`, guarded by a per-process bearer token), with a
  process restart as the fallback on older binaries.

In contrast, **AmneziaWG** (`internal/amneziawgnet/`) and **TUIC v5** (`internal/tuic/`) run as
**in-process native Go servers** without external child processes, bridging client traffic into
Xray-core via loopback SOCKS5 relays.

Servers and processes, all launched from `main.go`:

| Server / process | Package                           | Purpose                                                         | Default port      |
| ---------------- | --------------------------------- | --------------------------------------------------------------- | ----------------- |
| **Panel**        | `internal/web`                    | Admin REST/WS API + serves the embedded SPA                     | 2053              |
| **Xray-core**    | supervised via `internal/xray`    | The actual proxy engine; a child process, not Go code           | `inbounds[].port` |
| **mtg-multi**    | supervised via `internal/mtproto` | MTProto proxy child process for MTProto inbounds (multi-secret) | per inbound       |

Two key ideas that explain most of the complexity:

1. **The DB → Xray config pipeline.** Inbounds/clients live in the DB. On every change the
   backend regenerates the Xray config and applies it — preferring a _hot diff_ (live gRPC
   API mutation) over a full process restart. See §5.1.
2. **The Runtime abstraction.** Every state-changing inbound/client operation is dispatched
   through the `runtime.Runtime` interface. It is currently implemented only by **`Local`**
   (this box's Xray gRPC API), keeping services independent of the transport. See §5.2.

---

## 2. Tech stack

**Backend (Go 1.27):**

- Web framework: **Gin** (`gin-gonic/gin`) + sessions (cookie store), gzip.
- ORM: **GORM** with **SQLite** (default) or **PostgreSQL** (`XUI_DB_TYPE=postgres`).
- Scheduler: **robfig/cron/v3** (seconds-precision) for all background jobs.
- Xray: **xtls/xray-core** vendored as a library; the panel talks to the running core over
  its **gRPC API** and also shells out to manage the process.
- i18n: **nicksnyder/go-i18n**.
- Misc: gorilla/websocket, gopsutil (system stats), go-qrcode, gotp (2FA TOTP).

**Frontend (`frontend/`):**

- **React 19** + **Ant Design 6** + **Vite 8** + **TypeScript**.
- Data layer: **TanStack Query** (`@tanstack/react-query`) over the native **Fetch API**; **Zod 4** schemas.
- Router: **react-router 8**. Charts: **uPlot** (`frontend/src/components/viz/Sparkline.tsx`). Editor: **CodeMirror 6**.
- **Build output goes to `internal/web/dist/`** (see `vite.config.js` → `outDir`) and is
  embedded into the Go binary with `go:embed`. Two HTML entries: `index.html` (panel SPA)
  and `login.html`. The Go server serves the SPA; there is no separate frontend deployment.

**Important:** the legacy Go-template UI and `web/assets/` are **gone**. All HTML/JS comes
from the embedded Vite `dist/`. Don't look for `.html` templates in `internal/web`.

---

## 3. Request lifecycle (follow the data)

### 3.1 Admin API request (e.g. "add a client")

```
Browser (React, fetch)
  → POST {basePath}/panel/api/...
    → Gin engine (internal/web/web.go: initRouter)
      → middleware chain: SecurityHeaders → MaxBodyBytes (10 MiB; importDB exempt)
                          → [DomainValidator, if webDomain set] → gzip → sessions("3x-ui")
                          → base-path/cache-control context → Localizer
                          → API routes add: ConfigEnvelope (zstd + SHA-256) → CSRF
        → Controller (internal/web/controller/*.go)   // HTTP concerns only: bind, validate, respond
          → Service (internal/web/service/*.go)        // business logic + transactions
            → GORM → DB (internal/database)            // persistence
            → runtime.Runtime dispatch                 // apply to Xray (Local)
              → internal/xray (gRPC API or config regen + restart)
```

The controller layer is thin. **Business logic lives in services.** When something is wrong
with _behavior_, the bug is almost always in a service file, not a controller.

### 3.2 Share-link generation (panel export / QR)

```
Browser → GET {basePath}/panel/api/clients/links/:email
  → controller/client.go (getClientLinks)
    → service/inbound_sublink.go (GetAllClientLinks)
      → internal/sub/service.go (per-protocol share links)
        → reads inbounds+clients+hosts from DB; Host rows override address/SNI/path
```

### 3.3 Background work (cron jobs)

Scheduled in `internal/web/web.go` → `startTask()`. Each job is a struct in
`internal/web/job/`. Examples: poll Xray traffic every 5s, check client IP limits every 10s,
periodic traffic resets (hourly/daily/weekly/monthly). See §5.4.

---

## 4. Directory map (what lives where)

```
3x-ui/
├── main.go                     # Entry point: CLI (run / migrate / migrate-db / setting / cert),
│                               #   bootstrap, signal handling, restart loop
├── go.mod / go.sum             # Go deps (module path ends in /v3)
│
├── internal/                   # ALL backend Go code (private packages)
│   ├── config/                 # Env-var config: paths, DB kind/DSN, log level, version
│   │                           #   Every XUI_* env var is read here (config.go)
│   ├── database/
│   │   ├── db.go               # InitDB: connect, AutoMigrate, seeders (~1.4k lines). DB hotspot.
│   │   ├── migrate_data.go     # Data migrations (seeders/normalizers beyond AutoMigrate)
│   │   ├── dialect.go          # SQLite vs Postgres SQL differences
│   │   ├── dump_sqlite.go      # DB export/backup
│   │   └── model/              # **ALL GORM models** (model.go ~1.1k lines + siblings).
│   │                           #   ⭐ Start here for data shape.
│   ├── pia/                    # PIA WireGuard protocol client (auth, signed server list, /addKey)
│   ├── eventbus/               # In-process pub/sub (buffered channel): outbound.down|up,
│   │                           #   xray.crash, cpu.high, memory.high, login.attempt
│   ├── tunnelmonitor/          # Optional tunnel health probe (XUI_TUNNEL_HEALTH_* env vars):
│   │                           #   HTTP probe (default Cloudflare trace); repeated failures
│   │                           #   trigger an Xray restart hook. Independent of panel settings.
│   ├── xray/                   # Xray-core integration (the proxy engine wrapper)
│   │   ├── process.go          # Spawn/supervise the Xray child process (~750 lines)
│   │   ├── api.go              # gRPC client to a running Xray (add/remove user, stats) (~800 lines)
│   │   ├── hot_diff.go         # ⭐ Compute minimal live changes to avoid full restart (~500 lines)
│   │   ├── config.go           # Xray config object model
│   │   ├── inbound.go          # Inbound JSON shaping
│   │   ├── client_traffic.go   # ClientTraffic model (persisted as client_traffics)
│   │   ├── traffic.go          # Traffic type helpers
│   │   ├── log_writer.go       # Pipe Xray stdout/stderr into the panel logger
│   │   └── geodata/            # Browse geosite/geoip .dat: streaming protowire reader,
│   │                           #   cached category index, routing-token parsing (token.go)
│   │
│   ├── web/                    # The panel server
│   │   ├── web.go              # ⭐ Server bootstrap: initRouter (all routes) + startTask (all cron jobs)
│   │   ├── controller/         # HTTP handlers (thin). One file per resource:
│   │   │   ├── inbound.go      #   /panel/api/inbounds
│   │   │   ├── client.go       #   /panel/api/clients (CRUD + bulk + ips + onlines)
│   │   │   ├── group.go        #   client-group endpoints
│   │   │   ├── host.go         #   /panel/api/hosts   (per-inbound share-link host overrides)
│   │   │   ├── server.go       #   /panel/api/server  (status, xray version, certs, logs, DB import/export)
│   │   │   ├── setting.go      #   /panel/api/setting (settings + API tokens)
│   │   │   ├── xray_setting.go #   /panel/api/xray    (raw Xray config editor, WARP/Nord/PIA, geodata)
│   │   │   ├── api.go          #   /panel/api gateway (token auth, envelope + CSRF wiring)
│   │   │   ├── index.go        #   login/logout/csrf/2FA
│   │   │   ├── spa.go          #   SPA fallback for /panel UI routes
│   │   │   └── websocket.go    #   WS upgrade endpoint
│   │   ├── service/            # ⭐⭐ Business logic. This is where most real work happens.
│   │   │   ├── inbound.go              # Inbound CRUD core (~1.4k lines)
│   │   │   ├── inbound_traffic.go      # Per-client traffic accounting (~1.1k lines)
│   │   │   ├── inbound_clients.go      # Client-within-inbound operations
│   │   │   ├── inbound_sublink.go      # Client link-export helpers
│   │   │   ├── inbound_migration.go    # Inbound schema/format migrations
│   │   │   ├── client_crud.go          # Client create/read/update/delete
│   │   │   ├── client_bulk.go          # Bulk client ops (~1.6k lines)
│   │   │   ├── client_inbound_apply.go # ⭐ Apply client changes to the runtime (~1.2k lines)
│   │   │   ├── client_groups.go        # Client grouping
│   │   │   ├── client_link.go          # Per-client share-link generation
│   │   │   ├── client_external_link.go # External links attached to clients
│   │   │   ├── client_wireguard.go     # WireGuard client specifics
│   │   │   ├── client_paging.go        # Server-side pagination/sort/filter for client lists
│   │   │   ├── host.go                 # Host rows (share-link output overrides)
│   │   │   ├── server.go               # ServerService: status, certs, xray install, DB ops (~2.2k lines)
│   │   │   ├── setting.go              # SettingService: all panel settings + defaults (~1.3k lines)
│   │   │   ├── traffic_writer.go       # Batched persistence of traffic deltas to the DB
│   │   │   ├── xray.go                 # ⭐ XrayService: config gen + restart/hot-apply (~1.2k lines)
│   │   │   ├── xray_setting.go         # Raw Xray config persistence
│   │   │   ├── geodata.go              # Geo database browsing + routing-token validation
│   │   │   ├── xray_metrics.go         # Xray observability metrics
│   │   │   ├── metric_history.go       # Historical system/xray metrics
│   │   │   ├── reality_scan.go         # REALITY target scanner
│   │   │   ├── url_safety.go           # Outbound URL validation (SSRF guards)
│   │   │   ├── port_conflict.go        # Detect inbound port collisions
│   │   │   ├── fallback.go             # Xray fallback (SNI/ALPN routing on shared port)
│   │   │   ├── email/                  # Email notification service (SMTP)
│   │   │   ├── integration/            # External providers: warp.go, nord.go, pia.go
│   │   │   ├── outbound/               # Outbound config service
│   │   │   ├── panel/                  # Cross-cutting panel services:
│   │   │   │   ├── panel.go            #   panel-level helpers
│   │   │   │   ├── user.go             #   admin user auth (bcrypt)
│   │   │   │   ├── api_token.go        #   API token CRUD (SHA-256 hashed)
│   │   │   │   └── websocket.go        #   WS hub / push service
│   │   ├── runtime/            # ⭐⭐ The runtime abstraction (see §5.2)
│   │   │   ├── runtime.go      #   the Runtime interface (the contract)
│   │   │   ├── local.go        #   Local impl → this box's Xray gRPC API
│   │   │   └── manager.go      #   process-wide Local runtime holder
│   │   ├── job/               # Cron job structs (one file per job — see §5.4)
│   │   ├── middleware/        # Gin middleware: security.go (headers/HSTS), bodylimit.go,
│   │   │                      #   domainValidator.go, validate.go (CSRF), config_envelope.go
│   │   ├── global/           # Global singletons: web server + sub server handles, restart hook
│   │   ├── network/          # Custom net listeners (e.g. proxy-protocol aware)
│   │   ├── session/          # Session/cookie helpers
│   │   ├── websocket/        # WS hub implementation
│   │   ├── locale/ + translation/  # i18n middleware + 14 locale JSON catalogs
│   │   ├── entity/           # Shared request/response DTOs
│   │   └── dist/             # ⚠️ Vite build output, embedded via go:embed (generated — do not hand-edit)
│   │
│   ├── sub/                   # Share-link engine (no HTTP server)
│   │   ├── service.go         # ⭐ Per-protocol share-link builder
│   │   ├── host_sub.go        #   Host-row overrides applied to link output
│   │   ├── endpoint.go        #   Endpoint/protocol capability definitions
│   │   ├── vless_route.go     #   VLESS route shaping
│   │   ├── remark_vars.go     #   Remark variable expansion
│   │   └── links.go           #   Link helpers
│   │
│   ├── mtproto/              # Embedded MTProto (Telegram) proxy: manager.go + per-OS
│   │                         #   process supervision + orphan cleanup
│   ├── logger/              # App logger (op/go-logging + lumberjack rotation)
│   └── util/                # Leaf helpers (no business logic):
│       ├── common/          #   errors, misc
│       ├── crypto/          #   key/cert generation (x25519, ML-KEM/ML-DSA, ECH)
│       ├── link/            #   outbound share-link building primitives
│       ├── wirecodec/ + wireguard/  # WireGuard codec + integration helpers
│       └── random/, json_util/, reflect_util/, sys/, netproxy/, netsafe/, ldap/
│
├── frontend/                 # React SPA (built into internal/web/dist)
│   ├── vite.config.js        # ⭐ Build config: outDir → ../internal/web/dist, dev on :5173
│   │                         #   (strict) proxying to :2053, entries index/login.html
│   ├── package.json          # scripts: dev / build / preview / lint / typecheck / test / gen
│   └── src/
│       ├── main.tsx / routes.tsx / queryClient.ts   # SPA entry, router, query client
│       ├── entries/          # Extra HTML entry point: login.tsx
│       ├── pages/            # ⭐ Route screens. Mirrors the panel's feature areas:
│       │   ├── inbounds/     #   inbound list + the big inbound form (protocols/security/transport)
│       │   ├── clients/      #   client management screens
│       │   ├── hosts/        #   share-link host-override UI
│       │   ├── xray/         #   raw Xray config UI (routing, dns, outbounds, balancers, overrides)
│       │   ├── index/        #   dashboard/home
│       │   └── settings/, groups/, login/, api-docs/
│       ├── api/              # ⭐ Data layer: http-init, QueryProvider, queryKeys, websocket bridge
│       │   └── queries/      #   TanStack Query hooks (useStatusQuery, …)
│       ├── schemas/          # Zod schemas: protocols, forms, api, primitives
│       ├── generated/        # ⚠️ GENERATED from Go (see §5.5): schemas.ts, types.ts, zod.ts, examples.ts
│       ├── components/       # Reusable UI (clients/ form/ geodata/ ui/ viz/ feedback/ utility/)
│       ├── lib/              # Frontend domain logic (xray/ inbounds/ clients/)
│       ├── hooks/, models/, layouts/, i18n/, utils/, styles/
│       └── test/             # Vitest + golden fixtures (config-generation snapshot tests)
│
├── tools/openapigen/         # ⭐ Go program that emits frontend/src/generated/* from Go types (§5.5)
├── docs/                     # Markdown docs and the docs site
├── media/                    # README images
│
├── Dockerfile / docker-compose.yml / DockerEntrypoint.sh / DockerInit.sh   # Container build/run
├── install.sh / update.sh / x-ui.sh                        # VPS install + management CLI
├── x-ui.service.*  / x-ui.rc                               # systemd units (debian/rhel/arch) + rc script
├── windows_files/                                          # Windows service support
└── .github/workflows/        # CI: ci.yml, codeql.yml, docker.yml, release.yml,
                              #     mutation.yml, cleanup_caches.yml, claude-pr-review.yml,
                              #     claude-issue-analyst.yml
```

---

## 5. Cross-cutting subsystems (the parts that span many files)

### 5.1 DB → Xray config pipeline (config generation & application)

The panel never edits Xray's running config directly from controllers. The flow is:

1. A service mutates DB state (inbound/client/setting).
2. `XrayService` (`service/xray.go`) builds a fresh `xray.Config` from DB state
   (`GetXrayConfig`).
3. It tries a **hot apply** (`tryHotApply` → `xray/hot_diff.go`): diff old vs new config and
   push only the deltas over the Xray gRPC API (add/remove inbound, add/remove user) — **no
   process restart**, so live connections survive.
4. If the diff isn't hot-applicable (structural change), it falls back to a **full restart**
   of the Xray process (`xray/process.go`).

Restart is debounced via an atomic "need restart" flag (`SetToNeedRestart` /
`IsNeedRestartAndSetFalse`), consumed by a `@every 30s` cron task registered in `startTask()`
— any number of mutations inside the window causes at most one restart.

**Key files:** `service/xray.go` (orchestration), `xray/hot_diff.go` (the diff algorithm),
`xray/process.go` (process lifecycle), `xray/api.go` (gRPC calls), `xray/config.go` (config model).

Inbound-list statistics read only UUID, email and subscription ID from `clients`, joined
through `client_inbounds` in batches of up to 400 distinct inbound IDs. Full client records
remain the source for Xray and subscriptions. Missing shared-client traffic is fetched by
email, including rows owned by an inbound outside the requested list; enrichment preserves
the stored traffic, lifecycle and last-online fields.

### 5.2 Runtime abstraction — Local ⭐ most important

Every state-changing inbound/client operation goes through the `runtime.Runtime` interface so
services never talk to Xray directly. The interface has one implementation, `Local`, which
calls this box's Xray gRPC API. Keeping the indirection means a future transport can be added
without rewriting services.

- **Interface:** `internal/web/runtime/runtime.go` — `Name`, `AddInbound`, `DelInbound`,
  `UpdateInbound`, `AddUser`, `RemoveUser`, `UpdateUser`, `DeleteUser`, `AddClient`,
  `RestartXray`, `ResetClientTraffic`, `ResetInboundTraffic`, `ResetAllTraffics`.
- **`Local`** (`local.go`): calls this box's Xray gRPC API directly.
- **Dispatch:** `manager.go` holds the process-wide `Local` runtime. Services resolve it via
  `localRuntime()` / `runtimeFor(inbound)`; tests can swap it with
  `SetLocalRuntimeOverride`.

### 5.3 Traffic accounting

Per-client and per-inbound up/down counters originate from Xray's stats API and are persisted
to the DB. The Xray traffic job polls the core and writes deltas through the serialized traffic
writer so a reset and a poll cannot interleave.

**Key files:** `service/inbound_traffic.go`, `service/traffic_writer.go`,
`job/xray_traffic_job.go`, model `xray.ClientTraffic`.
Periodic resets: `job/periodic_traffic_reset_job.go` (keyed off `Inbound.TrafficReset`).

### 5.4 Background jobs (cron)

All registered in `web.go` → `startTask()`. Each is a struct with a `Run()` method in `internal/web/job/`:

| Schedule            | Job                                                                                              | Purpose / condition                                                                   |
| ------------------- | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------- |
| `@every 1s`         | `check_xray_running_job`                                                                         | Restart Xray if it died (2 consecutive down checks)                                   |
| `@every 30s`        | (inline func in `startTask`)                                                                     | Debounced Xray restart — consumes the "need restart" flag (§5.1)                      |
| `@every 5s`         | `xray_traffic_job`                                                                               | Pull traffic stats from Xray (5s start delay)                                         |
| `@every 10s`        | `check_client_ip_job`                                                                            | Record client IPs (30-min window) and enforce per-client IP limits                    |
| `@every 10s`        | `mtproto_job`                                                                                    | Reconcile `mtg` sidecars against enabled MTProto inbounds                             |
| `@every 10s`        | `amneziawg_job`                                                                                  | Reconcile embedded AmneziaWG interfaces against enabled local inbounds                |
| `@every 10m`        | `clear_logs_job` (`PruneXrayLogsJob`)                                                            | Truncate Xray access/error logs once either exceeds 64 MiB                            |
| `@hourly`           | `warp_ip_job`, `periodic_traffic_reset_job("hourly")`                                            | WARP IP rotation; traffic resets                                                      |
| `@daily`            | `clear_logs_job`, `periodic_traffic_reset_job("daily")`, `periodic_traffic_reset_job("monthly")` | IP-limit and Xray access/error log cleanup; daily resets and due monthly resets       |
| `@weekly`           | `periodic_traffic_reset_job("weekly")`                                                           | Weekly traffic resets                                                                 |
| default `@every 1m` | `ldap_sync_job`                                                                                  | Only if LDAP enabled; schedule configurable                                           |
| `@every 1m`         | `check_cpu_usage`                                                                                | Only if a CPU alarm is configured (email); publishes `cpu.high`       |
| `@every 1m`         | `check_memory_usage`                                                                             | Only if a memory alarm is configured (email); publishes `memory.high` |
| configurable        | `free_os_memory`                                                                                 | Only if `sys.MemoryReleaseIntervalMinutes() > 0`; returns heap to OS                  |

To change _when_ something runs, edit `startTask()`. To change _what_ it does, edit the job file.

### 5.5 Type generation (Go → TypeScript) ⚠️ don't hand-edit generated files

The Go backend is the schema source of truth. `tools/openapigen` (a Go program, with a
`StructAllow` allowlist of exported types) emits
`frontend/src/generated/{schemas,types,zod,examples}.ts`. The frontend build runs this first:

- `npm run gen:zod` → `go run ./tools/openapigen` (regenerate from Go)
- `npm run gen:api` → builds the OpenAPI doc (`scripts/build-openapi.mjs`, driven by the
  hand-maintained endpoint registry `src/pages/api-docs/endpoints.ts`)
- `npm run build` runs `gen:api` then `vite build`.

**Implication:** if you change a Go model/DTO that crosses the API boundary, regenerate the
frontend types (`cd frontend && npm run gen`) instead of editing `src/generated/` by hand.

### 5.6 Share-link generation

The panel's copy-link, QR, and client link-export paths all render through
`internal/sub/service.go`; `service/client_link.go` supplies per-client helpers and
`util/link/outbound.go` parses outbound links. **`Host` rows** (`model.Host`, edited under
/panel/api/hosts) override address/SNI/path/security per inbound in link output, applied in
`sub/host_sub.go`.

A malformed link for a specific protocol/transport combo (e.g. XHTTP + Reality) is usually a
field-lookup mismatch in **`internal/sub/service.go`** (and its tests / golden fixtures), or in
`util/link/outbound.go`. The frontend also has protocol schemas under
`frontend/src/schemas/protocols/` and `frontend/src/lib/xray/`.

### 5.7 Event bus (in-process pub/sub)

`internal/eventbus/` is a minimal buffered-channel pub/sub. Producers call a non-blocking
`Publish(Event)`; all subscribers receive every event. Event types: `outbound.down|up`,
`xray.crash`, `cpu.high`, `memory.high`, `login.attempt`, with structured payloads
(OutboundHealthData, LoginEventData, SystemMetricData). Producers include the CPU/memory
jobs and login handling; the email notifier
(`service/email/`) consumes them. Use it for cross-cutting notifications instead of importing
notification services into producers.

### 5.8 Tunnel health monitor

`internal/tunnelmonitor/` is an optional watchdog configured **only via env vars**
(`XUI_TUNNEL_HEALTH_*`, read in `internal/config/`), deliberately independent of panel
settings so it can be enabled from a systemd `EnvironmentFile` even when the panel is
unreachable. It periodically probes an HTTP URL (default: Cloudflare trace endpoint) through
the tunnel; after N successive failures (default 3) it fires a recovery callback wired to an
Xray restart.

---

## 6. Data model cheat-sheet

GORM models in `internal/database/model/` (main file `model.go` + siblings); all registered
for AutoMigrate in `internal/database/db.go`.

| Model                           | Table role                              | Notable fields                                                                                                                     |
| ------------------------------- | --------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| `User`                          | Admin login                             | bcrypt password, `LoginEpoch` (invalidates sessions)                                                                               |
| `Inbound`                       | An Xray inbound                         | `Tag` (unique), `Port`, `Protocol`, `Settings`/`StreamSettings`/`Sniffing` (JSON), `Enable`, `TrafficReset`, `ClientStats` (assoc) |
| `Client`                        | In-memory client view                   | UUID/email/flow/limits (parsed from inbound JSON; not persisted)                                                                   |
| `ClientRecord`                  | Persisted client (`clients`)            | `Email` (unique), `SubID`, `UUID`, `TotalGB`, `ExpiryTime`, `LimitIP`, `Group`, `Reset`                                            |
| `ClientGroup` / `ClientInbound` | Grouping + client↔inbound join          | many-to-many wiring, `FlowOverride`                                                                                                |
| `ClientExternalLink`            | Extra links attached to a client        | `Kind`, `Value`, `Remark`, `SortIndex`                                                                                             |
| `Host`                          | Share-link host overrides (per inbound) | `Address`, `Port`, `Sni`, `Path`, `Security`, `Fingerprint`, `SortOrder`, visibility/exclusion flags                               |
| `xray.ClientTraffic`            | Per-client counters (`client_traffics`) | `Email`, `Up`, `Down`, `Total`, `ExpiryTime`, `LastOnline`                                                                         |
| `InboundClientIps`              | IP set per client email                 | drives IP-limit enforcement                                                                                                        |
| `OutboundTraffics`              | Outbound counters                       | per outbound tag                                                                                                                   |
| `OutboundSubscription`          | Legacy provider-sub table               | Retained for DB compatibility; no panel UI or refresh job                                                                          |
| `Setting`                       | Key/value panel settings                | everything configurable                                                                                                            |
| `ApiToken`                      | REST API tokens                         | SHA-256 hash (plaintext shown once)                                                                                                |
| `InboundFallback`               | Fallback routing on a shared port       | SNI/ALPN/path → dest                                                                                                               |
| `HistoryOfSeeders`              | Seeder bookkeeping                      | prevents re-running one-off migrations                                                                                             |

---

## 7. Symptom → File index (start here when debugging)

| Symptom / task                                                   | Primary file(s)                                                             | Then check                                                                                          |
| ---------------------------------------------------------------- | --------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| Add/modify an **API endpoint**                                   | `controller/<resource>.go` (route registration at top of each file)         | corresponding `service/*.go`, `frontend/src/pages/api-docs/endpoints.ts`                            |
| **Inbound** create/update/delete behavior                        | `service/inbound.go`, `service/inbound_clients.go`                          | `runtime/*`, `service/xray.go`                                                                      |
| **Client** CRUD / limits / expiry                                | `service/client_crud.go`, `service/client_inbound_apply.go`                 | model `ClientRecord`, `service/inbound_traffic.go`                                                  |
| **Bulk** client operations slow/wrong                            | `service/client_bulk.go`                                                    | `service/client_paging.go`                                                                          |
| Xray **won't apply** a config change                             | `service/xray.go` (`RestartXray`, `tryHotApply`)                            | `xray/hot_diff.go`, `xray/process.go`                                                               |
| Xray **restarts when it shouldn't** (kills connections)          | `xray/hot_diff.go` (diff not classified as hot)                             | `service/xray.go`                                                                                   |
| **Traffic** counts wrong / reset behavior                        | `service/inbound_traffic.go`, `job/xray_traffic_job.go`                     | `service/traffic_writer.go`, `job/periodic_traffic_reset_job.go`                                    |
| **Share link / QR** malformed (per protocol)                     | `service/client_link.go`, `util/link/outbound.go`                           | `frontend/src/lib/xray/`, `frontend/src/schemas/protocols/`                                         |
| **Link export** output wrong (per protocol)                      | `internal/sub/service.go`                                                   | `sub/host_sub.go`, sub golden tests                                                                 |
| Share-link **host overrides** not applied                        | `service/host.go`, `sub/host_sub.go`                                        | model `Host`, `frontend/src/pages/hosts/`                                                           |
| **Settings** not saving / defaults                               | `service/setting.go`, `controller/setting.go`                               | model `Setting`                                                                                     |
| **Login / 2FA / sessions / CSRF**                                | `controller/index.go`, `service/panel/user.go`, `middleware/`               | `session/`                                                                                          |
| **API tokens**                                                   | `service/panel/api_token.go`, `controller/setting.go`                       | model `ApiToken`                                                                                    |
| **Port conflict** on inbound add                                 | `service/port_conflict.go`                                                  | `controller/inbound.go`                                                                             |
| **Fallbacks** (shared 443, SNI routing)                          | `service/fallback.go`, `controller/inbound.go`                              | model `InboundFallback`                                                                             |
| **Geo category browser** empty / won't open                      | `xray/geodata/` (`Store`, `reader.go`), `service/geodata.go`                | `controller/xray_setting.go` (`/panel/api/xray/geodata/*`), asset dir = `config.GetBinFolderPath()` |
| **`geosite:`/`geoip:` token** reported unknown in a routing rule | `xray/geodata/token.go`, `service/geodata.go` (`Validate`)                  | `frontend/src/lib/xray/geoTokens.ts`, `frontend/src/components/geodata/`                            |
| **Email notifications**                                          | `service/email/`                                                            | `internal/eventbus/` (consumers)                                                                    |
| **CPU / memory alerts** not firing                               | `job/check_cpu_usage.go`, `job/check_memory_usage.go`                       | `internal/eventbus/`, notifier settings in `service/setting.go`                                     |
| Xray auto-restart on **dead tunnel**                             | `internal/tunnelmonitor/`                                                   | `XUI_TUNNEL_HEALTH_*` in `internal/config/`                                                         |
| **WARP / Nord / PIA** outbound integration                       | `service/integration/warp.go` / `nord.go` / `pia.go`                        | `internal/pia/`, `frontend/src/pages/xray/overrides/`                                               |
| **MTProto** proxy issues                                         | `internal/mtproto/manager.go`, `mtproto/process*.go`                        | `job/mtproto_job.go`                                                                                |
| **DB migration** / new column                                    | `internal/database/db.go` (AutoMigrate list), `migrate_data.go`             | `model/model.go`                                                                                    |
| **Cron schedule** changes                                        | `web.go` → `startTask()`                                                    | the specific `job/*.go`                                                                             |
| **CORS / security headers / HTTPS**                              | `middleware/`, `web.go` (`initRouter`, TLS setup)                           | `config/` (env)                                                                                     |
| **Env vars / paths / DB type**                                   | `internal/config/config.go`                                                 | `.env.example`                                                                                      |
| **Frontend route / screen**                                      | `frontend/src/pages/<area>/`, `frontend/src/routes.tsx`                     | `frontend/src/api/queries/`                                                                         |
| **Frontend ↔ backend type mismatch**                             | regenerate: `cd frontend && npm run gen` (`tools/openapigen`)               | `frontend/src/generated/`                                                                           |
| **System status / CPU / metrics**                                | `service/server.go`, `service/xray_metrics.go`, `service/metric_history.go` | `controller/server.go`, gopsutil                                                                    |

---

## 8. Layering rules (where new code belongs)

1. **Controllers are thin.** Only: bind/validate input, call one service, shape the HTTP
   response. No DB queries, no Xray calls, no business rules in `controller/`.
2. **Services own the logic and transactions.** All business rules, DB access, and decisions
   about applying changes live in `service/`. If you're tempted to query GORM from a
   controller, move it to a service.
3. **Never touch Xray's running state from a controller or job directly.** Go through
   `XrayService` / the `runtime.Runtime` interface.
4. **Any state-changing inbound/client op must dispatch through `runtime.Runtime`**, not
   straight to `xray/api.go`.
5. **`internal/util/*` is leaf-only** (no imports of `service`/`controller`/`database`). Keep
   helpers pure.
6. **Don't hand-edit generated files:** `frontend/src/generated/*` and `internal/web/dist/*`.
   Regenerate instead.
7. **Models are the contract.** Changing a model field that crosses the API boundary means:
   update `model.go` → handle migration in `db.go`/`migrate_data.go` → regenerate frontend types.
8. **`internal/sub` is a link engine, not a server.** It renders share links on demand; do not
   add HTTP routes or panel state there.
9. **Cross-cutting notifications go through `internal/eventbus/`** — publish an event instead
   of importing the email service into producers.

---

## 9. Build / Test / Lint (verify your changes)

The canonical gate is the **Makefile** (mirrors CI): `make verify`. Also: `make gen`
(regenerate Zod/OpenAPI), `make lint` (Go + frontend), `make test` (Go `-shuffle=on` +
frontend), `make race`, `make build`. Run `make help` for everything. Raw commands:

**Backend (Go):**

```bash
go build ./...                      # compile everything
go test ./...                       # run all Go tests (many *_test.go alongside sources)
go test ./internal/web/service/...  # focused: service-layer tests
go test ./internal/xray/...         # hot-diff / process / api tests
go test ./internal/sub/...          # share-link + golden link tests
go vet ./...                        # static checks
golangci-lint run                   # full lint (gofumpt + goimports formatting)
go run main.go                      # run the panel locally (serves embedded dist if built)
```

**Frontend (`cd frontend`, Node 26 — see `.nvmrc`):**

```bash
npm install
npm run dev          # Vite dev server on :5173; proxies API to Go backend on :2053 (run `go run main.go` too)
npm run typecheck    # tsc --noEmit
npm run lint         # oxlint src
npm run test         # vitest (incl. golden config-generation snapshots)
npm run gen          # regenerate src/generated/* from Go (gen:zod + gen:api)
npm run build        # gen:api + vite build → outputs to internal/web/dist (then rebuild Go binary to embed)
```

**Full local loop:** `cd frontend && npm run build` (refresh embedded `dist/`) → back to repo
root → `go build ./...` / `go run main.go`.

**Docker:** `docker compose up -d` (uses `Dockerfile` + `DockerEntrypoint.sh`).

**CI** (`.github/workflows/`): `ci.yml` (build/test/lint), `codeql.yml` (security scan),
`mutation.yml` (mutation testing), `docker.yml` + `release.yml`
(multi-arch image + release builds), `cleanup_caches.yml`, `claude-pr-review.yml` (PR review
only - it changes no code), `claude-issue-analyst.yml` (issue triage). A maintainer re-runs
them with a comment starting `@claude review` on a PR or `@claude analyze` on an issue.

---

## 10. Gotchas & conventions

- **Module path is `.../v3`.** Internal imports use `github.com/mhsanaei/3x-ui/v3/internal/...`.
- **SQLite vs Postgres.** Default is SQLite at `{XUI_DB_FOLDER}/x-ui.db`. Postgres via
  `XUI_DB_TYPE=postgres` + `XUI_DB_DSN`. Some SQL paths are dialect-aware (`database/dialect.go`);
  test both when touching raw queries (there are `*_scale_postgres_test.go` suites).
- **`Inbound.Settings` / `StreamSettings` / `Sniffing` are raw JSON strings**, not structured
  columns. Parsing/validation happens in services and the `xray` package, not in GORM.
- **Hot-reload is the default; full restart is the fallback.** Changes that look config-only
  but cause a restart usually mean the diff in `xray/hot_diff.go` didn't recognize them as hot.
- **Restart is signal-driven.** `main.go` traps SIGHUP to restart the panel; the
  in-process restart hook (`global.SetRestartHook`) funnels into the same path.
- **i18n:** backend catalogs in `internal/web/translation/` (14 locales, shared with the
  frontend); frontend wiring in `frontend/src/i18n/`. Persian (`fa_IR`) is a first-class
  locale (Jalali calendar via `persian-calendar-suite`).
- **Tests live next to code** (`foo.go` ↔ `foo_test.go`), plus golden snapshots in
  `frontend/src/test/golden/fixtures/` for config generation — update fixtures intentionally,
  not blindly, when output changes.

## AmneziaWG outbound pseudo-protocol

The template stores `protocol: "amneziawg"` rows verbatim; Xray-core has no
such proxy. At config generation (`GetXrayConfig` and the outbound latency
probe's batch config) each row is swapped by `amneziawgnet.BuildSocksBridge`
into a loopback socks outbound pointed at the panel's egress server (port
`EgressBasePort`), authenticating with the row's tag as username. Sibling keys
(`mux`, `sendThrough`, `targetStrategy`, `streamSettings.sockopt`) survive the
swap. The embedded amneziawg-go client device lives in the panel process; an
unbridgeable entry (unreadable settings, empty/non-string tag) fails config
generation instead of skipping, because a skipped entry leaves
`protocol: "amneziawg"` behind -- which makes Xray refuse the whole config.

Traffic flow: Xray socks client -> egress SOCKS5 server (tag = username) ->
per-tag device netstack -> amneziawg-go tunnel. Domain targets are resolved by
a DNS exchange through that same netstack (`resolveTunnelVia`, default server
`DefaultTunnelDNSServer`), so names never leak to the panel host's resolver and
answers are valid at the tunnel's location; results cache for 60s. UDP flows
key sessions on the resolved address:port. Peer endpoints may be hostnames:
`resolvingBind.ParseEndpoint` resolves once at configure time (kernel
`wg setconf` semantics); a hostname whose DNS dies later needs a template
re-save or job restart to re-resolve.

`randomTrailers` defaults to false wherever the panel does not control the
peer (outbound form/schema): a receiver without 3.1 trailers silently drops
oversized packets from a sender with it enabled.
