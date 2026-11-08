# Catch me!

A minimal starting point for a hunter-versus-runner game: one Expo mobile app and one Go HTTP API.

## Structure

```text
app/                      Expo + React Native + TypeScript
  src/app/                Screens and layouts (Expo Router)
  src/api/                Backend requests and generated TypeScript client
backend/                  Go standard-library HTTP + Ent + SQLite
  api/openapi.json        Generated API document (do not edit)
  cmd/server/             Server entry point
  cmd/apigen/             API generation and freshness check
  cmd/dbgen/              Ent generation and freshness check
  internal/api/           Route registration, CORS, integration tests
  internal/controller/    HTTP parsing, response models, error mapping
  internal/service/       Game use cases and persistence interface
  internal/game/          Parsed domain values, lobby rules, playground geometry
  internal/db/            Ent-backed SQLite store
    ent/schema/           Handwritten Go table definitions
    ent/                  Generated database client (do not edit)
docs/game-spec.md         Game rules, milestones, and API overview
```

The app lets you select a playground on a map in the Android/iOS app (Google Maps on Android, Apple Maps on iOS), preview a north-aligned **6 × 6 grid**, and create a game. Cells default to approximately **2 km × 2 km** (12 km × 12 km total). Before saving, adjust **Cell side length (km)** between **0.1 and 5 km**, in 0.1 km steps; comma and dot decimals are accepted. The map fits the selected/resized grid, but panning or zooming never changes its physical size. The Go backend stores the center, boundary coordinates, chosen cell size, dimensions, and creation timestamp through Ent using the pure-Go `modernc.org/sqlite` driver. There is no separate database service; HTTP remains standard-library Go.

Creating a game returns one **join code per team** (runners and hunters). Anyone joins with a code and a display name, then waits in the **lobby**. There is no human gamemaster: the backend is the referee. When every player is ready and both teams have at least one player, a 10-second countdown starts and the game switches to running. The full rule set and the remaining milestones are in [docs/game-spec.md](docs/game-spec.md).

Saved sessions keep their stored size, including older 100 m-cell playgrounds. The Ent schema maps to the original `games` table and `session_id` primary key, preserving existing coordinates and timestamp text. Games created before join codes existed cannot be joined.

Implemented so far: playground setup, join codes, players, lobby, ready-up, countdown, and start. Runner routes, GPS location sharing, catching, and finishing are **not implemented yet**. Joining returns a secret player token that authenticates later requests; the app keeps it in memory only, so restarting the app loses the session. A session ID identifies saved configuration; it is **not** a credential or a join code. Anyone with the ID can read the playground (not the codes or players) in this local-development version. Games currently remain until the database is explicitly removed; no automatic cleanup is implemented yet.

The grid uses a local spherical approximation, not survey-grade geometry. Center latitudes are restricted to -80° through 80°, and playgrounds crossing the international date line are rejected. The frontend previews the geometry; the backend recomputes and stores the authoritative bounds. Individual cell boundaries are derived from those bounds rather than stored as 36 duplicate records. No map imagery or player location history is stored. Confirm the whole area is safe and permitted for outdoor play.

## Backend layers and Ent

Request flow is **API → controller → service → database**:

- `internal/api` registers endpoints and applies CORS/security headers. It contains no game validation or queries.
- `internal/controller` decodes bounded JSON requests, parses them into domain values (applying the default cell size when omitted), invokes the service, maps domain errors to HTTP statuses, and maps domain values to public response models.
- `internal/service` owns game creation/loading, joining, ready-up, leaving, random IDs and tokens, and timestamps. It accepts only parsed domain values and depends on a `GameRepository` interface, not SQLite or Ent. Lobby changes go through `UpdateLobbyBy*`, which load a game with its players, apply a domain change, and save it in one transaction.
- `internal/db` implements that interface using generated Ent setters and lookups. Loaded rows are parsed again, so invalid stored data becomes a server error. Ent-specific errors/entities stay inside this layer.
- `internal/game` holds domain values and geometry with no HTTP/database dependencies. Values are parsed, not validated: `ParseSessionID`, `ParseCoordinates`, `ParseCellSize`, and `NewPlayground` are the only way to build them from raw input, and their fields are private. The Ent schema reuses these parsers as validators. `cmd/server` wires the layers together; no dependency-injection framework or generic repository hierarchy is used.

The public API contract is separate from database entities. Changing an Ent model must not accidentally change frontend responses or expose future private fields.

### Declare and query tables in Go

Edit **`backend/internal/db/ent/schema/game.go`** for table/field definitions, then regenerate:

```sh
# From backend/:
go run ./cmd/dbgen
go run ./cmd/dbgen -check
go test ./...
```

Or from the repository root:

```sh
docker compose run --rm --workdir /workspace/backend api-tools go run ./cmd/dbgen
docker compose run --rm --workdir /workspace/backend api-tools go run ./cmd/dbgen -check
```

Ent **v0.14.6** is pinned in `go.mod`; the generator uses that same version. Keep generated files under `internal/db/ent/` in Git; never edit them manually (the `schema/` subdirectory is handwritten input). `-check` generates into a temporary directory and compares changed, missing, and extra files without updating committed output. Go tests and the backend Docker build include this freshness check, so CI catches stale Ent code automatically. API generation remains a separate command, needed only when the HTTP contract changes.

Schema validators protect writes through Ent; they are not a substitute for domain parsing, authorization, or a complete set of SQLite CHECK constraints. The `players` table stores only a SHA-256 hash of each player token.

### Schema initialization and existing databases

Startup uses Ent's automatic schema migration with column/index dropping explicitly disabled. It creates the table on a fresh database and reconciles the mapped schema on an existing one. Tests seed the exact pre-Ent schema and verify that old sessions and new writes survive repeated startup without changed IDs, sizes, coordinates, or timestamp precision. The SQLite driver remains pure Go and the API still builds with `CGO_ENABLED=0`.

This is a **local-development migration policy**, not a production migration workflow. Back up existing databases before schema changes. Before production, adopt reviewed, versioned migrations rather than automatically applying evolving schemas during server startup; some schema changes can rebuild SQLite tables even with dropping disabled. Ent includes migration dependencies, but no Atlas CLI or additional running service is required for this setup.

## Go-first API generation and frontend service

The **Go source is the source of truth**:

```text
Go response structs + handler annotations
        ↓ Swaggo (development tool)
backend/api/openapi.json
        ↓ @hey-api/openapi-ts
app/src/api/generated/
```

Response fields come from `backend/internal/controller/models.go` and its JSON tags. Endpoint parameters and response codes come from the annotations on controller methods in `game.go` and `health.go`; actual routing is registered in `internal/api/handler.go`. Keep route registrations and `@Router`, `@Param`, and `@Success`/`@Failure` annotations synchronized; Go's standard library cannot infer an HTTP contract from arbitrary code.

Both the API document and frontend client are autogenerated. **Do not edit either output by hand.** The stable Swaggo tool emits Swagger/OpenAPI **2.0** JSON (identified by `"swagger": "2.0"`), despite the `openapi.json` filename. The frontend generator supports this format. We deliberately avoid a prerelease OpenAPI 3 generator for this simple setup.

Swaggo is pinned and invoked as an external development tool. It does not add requirements to `backend/go.mod`, generate a Go runtime docs package, or introduce a third-party server dependency. Generated client code uses fetch and is bundled into the app.

The Go annotation `@ID` becomes `operationId` and determines each generated service function:

| Endpoint | Generated function | Purpose |
| --- | --- | --- |
| `GET /health` | `getHealth()` | Backend health check |
| `POST /api/games` | `createGame()` | Save a playground; return the game and both join codes |
| `GET /api/games/{sessionId}` | `getGame()` | Load a saved playground (no codes or players) |
| `POST /api/players` | `joinGame()` | Join with `{code, name}`; return a player token and the game state |
| `GET /api/me/game` | `getGameState()` | Game state as seen by the authenticated player |
| `PUT /api/me/ready` | `setReady()` | Set `{ready}`; return the game state |
| `DELETE /api/me` | `leaveGame()` | Leave before the game starts |

The `/api/me` endpoints require `Authorization: Bearer <playerToken>`. The game state contains only what the requesting player may see, for example only their own team's join code. Errors: 400 invalid input, 401 missing/unknown token, 404 unknown game or code, 409 rule violation (already started, full, name taken).

Create a game by posting a JSON center to `/api/games`:

```json
{
  "latitude": 52.52,
  "longitude": 13.405,
  "cellSizeMeters": 2000
}
```

The response is HTTP **201**, with `game` (`sessionId`, `createdAt`, and a `playground` containing `center`, `bounds` (`north`, `south`, `east`, `west`), `rows`, `columns`, and `cellSizeMeters`) and `joinCodes` (`runner`, `hunter`).

`cellSizeMeters` is an optional integer: omitted/null defaults to 2000. Explicit values must be 100–5000 in steps of 100. The backend rejects invalid sizes and computes bounds from the requested center and size rather than trusting frontend geometry.

The handwritten `app/src/api/client.ts` wrapper supplies the API URL, a five-second timeout, readable errors, and basic runtime response validation. Its types and HTTP request functions come from generated code, not manually duplicated interfaces or URLs. Creation is not automatically retried: if the network drops after a save, an unreferenced session may remain in the database.

You can call the generated service directly too:

```ts
import { API_URL } from '@/api/client';
import { createGame } from '@/api/generated';

const { data } = await createGame({
  baseUrl: API_URL,
  body: { latitude: 52.52, longitude: 13.405, cellSizeMeters: 2000 },
  throwOnError: true,
});
console.log(data.sessionId);
```

### Change or add an endpoint

1. Change the Go response/request structs, handler, and its annotations.
2. From `app/`, run `npm run api:update` (requires local Go and Node).
3. Run `npm run api:check`, `npm run typecheck`, `npm run lint`, `npm run test:api`, and the Go tests.
4. Include both regenerated outputs with your changes.

Or update and check using Docker, without local Go or Node:

```sh
docker compose run --rm --build api-tools npm run api:update
docker compose run --rm api-tools npm run api:check
```

The `api-tools` service is an on-demand development container; it does not start with the app. It writes generated files into the mounted backend and app source directories and caches Go tool downloads. The regular Expo container has no Go toolchain.

| Command (from `app/`) | What it does |
| --- | --- |
| `npm run api:update` | Generate the API document from Go, then regenerate the frontend client |
| `npm run api:check` | Check both generated outputs against their sources, without modifying them |
| `npm run api:generate` | Regenerate only the frontend client from the current document |
| `npm run api:check-client` | Check only the frontend client, without requiring Go |

From `backend/`, `go run ./cmd/apigen` updates just the API document; add `-check` to test freshness. The initial tool invocation downloads the pinned Swaggo tool and its development dependencies, so it needs network access. Later invocations use Go's cache.

Frontend tests check generated-file freshness, the stale-file detector, JSON requests, bearer tokens, successful responses, errors with HTTP status, runtime guards, timeouts, the server-clock countdown, and grid geometry/labels. Backend tests cover both generation pipelines, domain parsing, lobby rules (countdown start/cancel, joining limits, actions after start), isolated service behavior, real SQLite persistence and legacy compatibility, HTTP save/load and lobby flows, authentication, invalid requests, storage errors, CORS, and grid geometry. Contract checks are smoke tests, **not full OpenAPI schema validation**.

The backend Docker build runs the source/document freshness test. The app Docker build regenerates the client from that document. Update the generated files before rebuilding after an API change.

## Run with Docker (recommended)

Install and start **Docker Desktop** with Linux containers and Docker Compose v2. You do not need Go or Node installed on your computer for this workflow.

From the repository root:

```sh
docker compose up --build
```

- **API:** <http://localhost:8080/health>
- **App browser preview:** <http://localhost:8081>

The browser preview intentionally has no embedded map; use Android/iOS to create a game. Joining and the lobby work in the browser too.

The backend image compiles a static Go executable, runs tests and vet during the build, and runs the executable as a non-root user. Compose waits for its health check before starting Expo.

SQLite is stored at `/data/catch-me.sqlite` in the `game_data` Docker volume. Container restarts/recreation and `docker compose down` preserve saved games. **Do not use `docker compose down -v` unless you intend to delete all volume data.** Back up the database with the backend stopped to avoid an inconsistent copy. Direct `docker run` users must also mount a persistent volume at `/data`.

The app container is an **Expo development server**, not a production frontend or an Android/iOS emulator. It uses Node 24 and mounts source files for live reload without sharing Windows `node_modules` with Linux. The native app still runs in Expo Go or a development build on your device. Rebuild the app image after changing dependencies or other files not mounted in `compose.yaml`.

`tsconfig.json` stays writable inside the app image because Expo updates it during startup; it is deliberately not mounted as an individual file. Rebuild after changing the host's TypeScript configuration. Both running services have readiness health checks.

### Use a physical phone with Docker

Copy the **root** `.env.example` to `.env` (PowerShell):

```powershell
Copy-Item .env.example .env
```

Replace the three values with your computer's LAN IP, for example:

```dotenv
EXPO_HOST=192.168.1.100
EXPO_PUBLIC_API_URL=http://192.168.1.100:8080
WEB_ORIGIN=http://192.168.1.100:8081
```

Restart with `docker compose up --build`. Open `exp://192.168.1.100:8081` in Expo Go, or scan the QR code shown in the Expo logs. Your phone and computer must share a network, and ports **8080 and 8081** must be allowed through the firewall on a trusted private network. Keep the browser preview on `http://192.168.1.100:8081` too, so its origin matches `WEB_ORIGIN`.

`EXPO_HOST` overrides the hostname Expo advertises, so the phone gets the computer's IP rather than the container's private IP. For an Android Studio emulator, use `EXPO_HOST=10.0.2.2` and `EXPO_PUBLIC_API_URL=http://10.0.2.2:8080`.

Some Expo Go environments require Expo CLI and Expo Go to be signed in to the same Expo account. Sign in inside the container with `docker compose exec app npx expo login`; its CLI state is kept in the `expo_state` volume. Do not bake credentials into images.

Compose takes its configuration from the **root `.env`** and passes the API URL to Expo. `app/.env.local` is for the non-Docker workflow only; dotenv loading is disabled inside the container to avoid conflicting settings.

### Container commands

```sh
# Stop containers (keeps Expo login state).
docker compose down

# View logs.
docker compose logs -f

# Run frontend checks in the running container.
docker compose exec app npm run typecheck
docker compose exec app npm run lint
docker compose exec app npm run test:api

# Run backend checks via the build's test stage, without local Go.
docker build --target test ./backend
```

Backend source changes require `docker compose up --build` again. The Expo development server must not be exposed publicly; this Compose setup is for local development, not a production deployment.

## Requirements for running without Docker

- **Node.js 24 LTS**, version 24.3 or newer, and npm. Node 25 is not supported by this Expo SDK.
- **Go 1.26 or newer**, available on your PATH. Install a currently supported release from <https://go.dev/dl/>. Initial builds download the pinned SQLite driver and its transitive dependencies.
- Expo Go matching SDK 58 for testing on a phone, or a compatible development build.

## Run locally

### 1. Start the backend

In a terminal, from the repository root:

```sh
cd backend
go run ./cmd/server
```

The API listens on port **8080**, on all network interfaces so a phone on your network can reach it. Open <http://localhost:8080/health> to check it:

```json
{"status":"ok","service":"catch-me-backend"}
```

Optional flags:

```sh
go run ./cmd/server -addr :8080 -web-origin http://localhost:8081
```

`-web-origin` allows the optional Expo browser preview to call the API. Native app requests do not need CORS. CORS is not authentication. This plain-HTTP starter is for trusted local development; use HTTPS and add authentication before deploying gameplay or location data.

The local database defaults to `backend/catch-me.sqlite` when run from `backend/` and is Git-ignored. Use `-db PATH` to select a different file; its parent directory must already exist.

### 2. Configure the app's API address

From the repository root in another terminal (PowerShell):

```powershell
cd app
Copy-Item .env.example .env.local
```

For macOS/Linux, use `cp .env.example .env.local` instead. Set `EXPO_PUBLIC_API_URL` in `app/.env.local` based on where the app runs:

| Device | API address |
| --- | --- |
| Browser on your computer | `http://localhost:8080` |
| Android Studio emulator | `http://10.0.2.2:8080` |
| iOS simulator on the backend computer | `http://localhost:8080` |
| Physical phone | `http://YOUR_COMPUTER_LAN_IP:8080` |

On a physical phone, `localhost` refers to the phone, **not** your computer. Keep the phone and computer on the same Wi-Fi, and allow the backend's port through your firewall on a trusted private network. On Windows, use `ipconfig` to find your active adapter's IPv4 address. An Expo tunnel only tunnels the Expo development server, not this Go API.

`EXPO_PUBLIC_` values are embedded in the app: never put secrets there. Fully reload the app after changing the URL. Local HTTP may be blocked in standalone native builds; use an HTTPS API for those builds.

### 3. Start Expo

In the `app/` terminal:

```sh
npm ci
npm start
```

On iPhone, Expo Go is enough: iOS uses Apple Maps, which needs no API key. Android Google Maps with our own key requires a native build configured as described below. Pan/zoom the map, tap to choose a center, and press **Create game**. The app shows the runner and hunter join codes; enter your name and join a team yourself. Other players enter a code and their name on the home screen. In the lobby, everyone taps **Ready**; the game starts 10 seconds after the last player is ready if both teams have players.

For a browser preview:

```sh
npm run web
```

Browser preview uses port **8081** to match the backend's default allowed web origin. iOS simulator development requires macOS; a native iPhone build can be produced using macOS or EAS, then connect to the development server running on Windows.

## Maps setup

`react-native-maps` was installed using `npx expo install react-native-maps`, selecting the SDK 58-compatible version. Android uses `PROVIDER_GOOGLE`; iOS uses the default Apple Maps provider, which works in Expo Go and needs no API key. The grid is drawn with polygons and polylines. Manual selection requests no GPS permission and does not track the device.

For Android Google Maps with our own key, use a native build:

1. Create a Google Cloud project, configure billing, and enable **Maps SDK for Android**. Check current Google pricing and configure budget alerts.
2. Create a key restricted to package `com.catchme.app` plus the build's signing SHA-1, and to **Maps SDK for Android**. This is an initial app identifier; choose your final unique identifier before publishing and update key restrictions accordingly.
3. Set `GOOGLE_MAPS_ANDROID_API_KEY` in `app/.env.local` for local work, or in the root `.env` for Docker Expo. `app/app.config.js` passes it to the maps config plugin. Never commit real keys. Native SDK keys are embedded in binaries and must be protected with application/API restrictions, not treated as secret server credentials.
4. Build locally with `npx expo run:android`. This generates the native project from Expo config; do not hand-edit generated native directories. Alternatively configure EAS and supply the key through its build environment (EAS credentials/project setup is not included in this milestone).
5. Rebuild the native binary after changing keys or native configuration. Metro reloads and JavaScript exports do not update native configuration.

Use the appropriate signing SHA-1 for each Android build (development and Play signing certificates can differ). Missing/restricted keys can produce a blank map; they cannot be validated by TypeScript tests or CI bundle exports. Use an HTTPS backend if the native build blocks local HTTP; certificate verification must remain enabled.

See the [SDK 58 maps setup documentation](https://docs.expo.dev/versions/v58.0.0/sdk/map-view/) for provider and signing-key instructions.

## GitHub Actions

`.github/workflows/ci.yml` runs a single **Project checks** job on pull requests, pushes to `main`, and manual runs from GitHub's Actions tab.

It checks Go formatting, backend tests and vet, generated API freshness, frontend TypeScript and lint, API client/generation tests, and Android/iOS/web JavaScript bundle exports. Checks do not regenerate committed files; stale generated output fails CI and must be updated with `api:update` before pushing.

The workflow uses Node 24 and Go 1.26, caches npm downloads and Go modules/builds, and requires no custom secrets. Actions are pinned to commit hashes, the repository token is read-only, and superseded runs are canceled.

This is validation only: it does not deploy containers, publish images, produce signed native app binaries, or submit to app stores. GitHub-hosted execution starts after the workflow is committed and pushed to GitHub. To block merges on failures, configure a branch protection/ruleset requiring **Project checks** after its first run.

## Checks

Run in `app/`:

```sh
npm run typecheck
npm run lint
npm run test:api
npm run api:check
npx expo-doctor
```

Run in `backend/`:

```sh
go test ./...
go vet ./...
gofmt -l .
```

Use `npx expo install <package>` when adding app packages, so Expo selects compatible versions. Commit `app/package-lock.json` for reproducible installs. Do not use `npm audit fix --force` blindly: it can replace Expo with an incompatible SDK.
