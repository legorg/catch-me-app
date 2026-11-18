# Catch me!

A minimal starting point for a hunter-versus-runner game: one Expo mobile app and one Go HTTP API.

## Structure

```text
app/                      Expo + React Native + TypeScript
  src/app/                Screens and layouts (Expo Router)
  src/api/                Backend requests and generated TypeScript client
  src/i18n/               App texts per language (en.json, de.json) and translate()
backend/                  Go standard-library HTTP + GORM + SQLite
  api/openapi.json        Generated API document (do not edit)
  cmd/server/             Server entry point
  api/                    Route registration, security headers, integration tests
  controller/             HTTP parsing, response models, error mapping
  service/                Game use cases and persistence interface
  game/                   Parsed domain values, lobby rules, playground geometry
  db/store.go             GORM table structs and SQLite store
docs/game-spec.md         Game rules, milestones, and API overview
```

The app lets you select a playground on a map in the Android/iOS app (Google Maps on Android, Apple Maps on iOS), preview a north-aligned **6 × 6 grid**, and create a game. Cells default to approximately **2 km × 2 km** (12 km × 12 km total). Before saving, adjust **Cell side length (km)** between **0.1 and 5 km**, in 0.1 km steps; comma and dot decimals are accepted. The map fits the selected/resized grid, but panning or zooming never changes its physical size. The Go backend stores the center, chosen cell size, and creation timestamp through GORM using the pure-Go `modernc.org/sqlite` driver. There is no separate database service; HTTP remains standard-library Go.

Creating a game returns one **join code per team** (runners and hunters). Anyone joins with a code and a display name, then waits in the **lobby**. Hunters then join or create a **hunter team** by name; hunters of one team see each other's positions, already in the lobby, while other hunter teams never do. There is no human gamemaster: the backend is the referee. When every player is ready (hunters need a hunter team) and both teams have at least one player, a 10-second countdown starts and the game switches to running. The full rule set and the remaining milestones are in [docs/game-spec.md](docs/game-spec.md).

**Languages:** the app starts in the device language if it is supported (otherwise English), and a switch on the screen changes it until the app restarts. To add a language, copy `app/src/i18n/en.json` to `<code>.json`, translate every text (keep `{placeholders}`), and add it to `LANGUAGES` in `app/src/i18n/index.ts`; the typecheck and `npm run test:api` report missing texts. Backend error messages stay English.

**Spectators (converted runners implemented):** authenticated game-state responses include `role: player | spectator`, separate from the original runner/hunter `team`. A caught/finished runner automatically becomes a read-only spectator, derived from the persisted outcome. Only while the game is running, that viewer receives the latest real reports of all runners and all hunter teams, without reveal delays or fake/zone-only concealment. The game screen switches to a spectator map: **blue runners**, **red hunters**, **black × catches**, and **blue ✓ frozen finish reports**, including the viewer's own retired report. Tap a pin for the name, team/status, report time and accuracy; hunter pins also name their hunter team. Missing/expired reports have no marker. The HUD shows the spectator role, own outcome and a legend; Info explains report age/frozen positions and fair play. Route markers, joker snapshots, reveals, zone overlays and sketches are hidden in this view. There are no gameplay controls, own-location dot/locate button or permission prompts; foreground/background sharing stays stopped, including on session restore. Active-player visibility is unchanged, especially finish-location privacy. Spectators use the existing automatic result navigation on game end. Direct spectator joining, timeline and full history remain unimplemented. Outcomes, end conditions and retention are unchanged; no database reset or migration is needed.

Implemented so far: playground setup, join codes, players, lobby, ready-up, countdown, start, runner routes (one shared route per game from a landmark on the southern edge row to one on the northern edge row, regenerable twice per game by any runner while no runner is ready, visible to all runners), and foreground location sharing. Runners must stand within 50 m of the start landmark to get ready; while running, teammates see each other and hunters see the runners' positions every 30 seconds. Fairness-based self-capture and explicit finish confirmation are implemented. An eligible runner sees a flag icon for "Target reached" (shared Ready/finish location check: 50 m, report at most 30 seconds old). After confirmation, sharing stops and the runner can keep watching without revealing the finish location to hunters. **Automatic game end and a basic result screen are implemented:** the last runner's catch/finish atomically records `endedAt` and changes the game state to `ended`; the server then rejects all further position reports and gameplay actions. When the app learns of the end, it stops sharing for everyone, including hunters, and replaces gameplay (including an open joker modal) with the result screen. The same result is accessible by full game ID from the start screen or a direct link, without joining, coordinates or player credentials. Maps and movement history are not implemented yet. Positions are shared in the foreground before the start; while running, players who allow "Always" access keep sharing in the background (installed app builds only, not Expo Go; it stops when leaving the game screen or closing the app). Joining returns a secret player token that authenticates later requests; it is mirrored to native secure storage to resume the session. Restoring an ended session opens its result and clears the completed game's token. A session ID is not a player credential or a join code, but is a **confidential result access key**: anyone knowing it can view names and outcomes after the game ends. Games currently remain until the database is explicitly removed.

**Positions:** every reported position is a row in the `positions` table. A player's newest row is their current position; a runner's reveal for hunters is their last row before the reveal time. All rows together are the history for a replay or chart after the game, which nothing reads yet (no endpoint, no chart). Positions are deleted when a player leaves, and automatically **7 days after the game's start** (or creation, if it never started): at server startup and then hourly.

The grid uses a local spherical approximation, not survey-grade geometry. Center latitudes are restricted to -80° through 80°, and playgrounds crossing the international date line are rejected. The frontend previews the geometry; the backend recomputes the authoritative bounds from center and cell size. No map imagery is stored. Confirm the whole area is safe and permitted for outdoor play.

## Backend layers and database

Request flow is **API → controller → service → database**:

- `api` registers endpoints and applies security headers. It contains no game validation or queries.
- `controller` decodes bounded JSON requests, parses them into domain values (applying the default cell size when omitted), invokes the service, maps domain errors to HTTP statuses, and maps domain values to public response models.
- `service` owns game creation, joining, ready-up, leaving, history cleanup, random IDs and tokens, and timestamps. It accepts only parsed domain values and depends on a `GameRepository` interface, not SQLite or GORM. Lobby changes go through `UpdateLobbyBy*`, which load a game with its players, apply a domain change, and save it in one transaction.
- `db` implements that interface with GORM. Loaded rows are parsed again, so invalid stored data becomes a server error. GORM types and errors stay inside this layer.
- `game` holds domain values and geometry with no HTTP/database dependencies. Values are parsed, not validated: `ParseSessionID`, `ParseCoordinates`, `ParseCellSize`, and `NewPlayground` are the only way to build them from raw input, and their fields are private. Optional values (route, position, reveal) are plain values whose zero value means "none". `cmd/server` wires the layers together; no dependency-injection framework or generic repository hierarchy is used.

The public API contract is separate from database rows. Changing a table must not accidentally change frontend responses.

### Tables

The schema is the table structs at the top of **`backend/db/store.go`** (`games`, `players`, `runners`, `positions`, `jokers`, `sketches`). There is no code generation. Startup runs GORM's `AutoMigrate`, which creates missing tables, columns, and indexes but never drops existing columns. Shared player data stays in `players`; only runners have a one-to-one `runners` row with a single `active`/`caught`/`finished` status and common `ended_at`. Players reference their game, and runner states and positions their player, with `ON DELETE CASCADE`. Positions are only inserted, never updated. The `players` table stores only a SHA-256 hash of each player token. Times are stored as UTC text in SQLite's sortable format. GORM's SQL logging is off, because it would contain player positions.

The runner-state refactoring targets fresh development databases: there is no migration from the old `players.caught_at` / `finished_at` fields. AutoMigrate does not transfer that data or remove old columns; existing games without `runners` rows are incompatible. Local databases are never reset automatically; obtain explicit authorization before deleting saved data. Each runner in the public API has `runner: {status, endedAt?, caughtPosition?}`; hunters have no `runner` object. Status is `active`, `caught`, or `finished`; only caught runners can expose a catch position, and neither active nor finished runners do. The old top-level `caughtAt`, `finishedAt`, and `caughtPosition` fields are removed. Backend and app must use the matching API/client version.

The SQLite driver remains pure Go (`modernc.org/sqlite`, plugged into GORM's official SQLite dialect) and the API still builds with `CGO_ENABLED=0`. GORM's dialect also pulls in the cgo driver `mattn/go-sqlite3` as a dependency; it is never used.

This is a **local-development migration policy**, not a production migration workflow. Databases from the earlier Ent version are **not compatible**; delete them (`docker compose down -v` removes the `game_data` volume and all saved games). Before production, adopt reviewed, versioned migrations rather than automatically applying evolving schemas during server startup.

## Go-first API generation and frontend service

The **Go source is the source of truth**:

```text
Go request/response structs + operation table (backend/api/spec.go)
        ↓ swaggest/openapi-go (go generate ./cmd/server)
backend/api/openapi.json
        ↓ @hey-api/openapi-ts
app/src/api/generated/
```

Request and response fields come from structs in `backend/controller/` (`models.go`, and `result.go` for public results) and their tags: `json` for names, `path` for URL parameters, `validate` for constraints (min/max/length/oneof, mirrored into the schema), and `example`, `minimum`, `maximum`, `pattern`, `default`, `enum` where more detail is needed. Fields are required unless they have `omitempty`. Endpoint summaries, parameters and response codes are listed in the operation table in `backend/api/spec.go`; actual routing is registered in `api/handler.go`. Keep the route registrations and the operation table synchronized; Go's standard library cannot infer an HTTP contract from arbitrary code.

Both the API document and frontend client are autogenerated. **Do not edit either output by hand.** The document is OpenAPI **3.1** JSON (`"openapi": "3.1.0"`). `GameStateResponse` is a `oneOf` of the runner and hunter variants, discriminated by `team`.

The generator is a library (`swaggest/openapi-go`) and a small command (`backend/cmd/openapi`), so it is a normal dependency of `backend/go.mod` (the spec builder lives in the `api` package and is linked into the server, but only the generator command calls it). Generated client code uses fetch and is bundled into the app.

The operation `id` in `backend/api/spec.go` becomes `operationId` and determines each generated service function:

| Endpoint | Generated function | Purpose |
| --- | --- | --- |
| `GET /health` | `getHealth()` | Backend health check |
| `POST /api/games` | `createGame()` | Save a playground; return the game and both join codes |
| `GET /api/games/{sessionId}/result` | `getGameResult()` | Public, coordinate-free result of an ended game, using the full game ID |
| `POST /api/players` | `joinGame()` | Join with `{code, name}`; return a player token and the game state |
| `GET /api/me/game` | `getGameState()` | Game state as seen by the authenticated player |
| `PUT /api/me/ready` | `setReady()` | Set `{ready}`; return the game state |
| `DELETE /api/me` | `leaveGame()` | Leave before the game starts |
| `POST /api/me/route` | `regenerateRoute()` | Give the game a new route (any runner, 2 times per game, while no runner is ready) |
| `PUT /api/me/position` | `updatePosition()` | Report the current position; the server applies the visibility rules |

The `/api/me` endpoints require `Authorization: Bearer <playerToken>`. The game state contains only what the requesting player may see, for example positions by team and visibility rules (both join codes are visible to every player). Errors: 400 invalid input, 401 missing/unknown token, 404 unknown game or code, 409 rule violation (already started, full, name taken).

**Results by confidential game ID:** `GET /api/games/{sessionId}/result` requires no token or joining. Use the complete 32-character lowercase hexadecimal game ID, not a six-digit join code. The response contains `sessionId`, `startsAt`, `endedAt`, `durationSeconds` (whole seconds, rounded down), and `players` in join order. Each player has `playerId`, `name`, and `team`; hunters additionally have their chosen `hunterTeam`, runners have `runner: {status, endedAt}` (`caught` or `finished`). No route, positions, history, join codes, tokens or hashes are returned in this first version. Unknown and unfinished games return the same `404`; invalid IDs return `400`. Results are not cached (`Cache-Control: no-store`) and remain available after position retention expires, while the game stays stored.

**In the app:** choose **View game result** on the start screen and paste the full ID, or open `/results/{sessionId}` (installed-app deep link: `catch-me://results/{sessionId}`). Whitespace and uppercase hex are normalized. The screen shows start/end, duration (minutes:seconds), runner outcomes/times and hunter teams, with loading/error/retry states and a return-to-start button. It requests neither joining nor location permission. Automatic game-end navigation opens this same screen, including after restoring a completed session, and clears that completed session so it is not resumed again. Save the selectable ID somewhere private if you want to reopen the result later. Knowing the ID grants access; it is not a public listing. The agreed later extension is every player's retained in-game coordinates/timestamps for a post-game history using this same ID-based access rule; this is not implemented yet and does not change position retention. The app's development result-request log redacts IDs; backend/proxy access logs still need review before production.

```sh
curl http://localhost:8080/api/games/00112233445566778899aabbccddeeff/result
```

Replace the example ID with an actual ended game's `sessionId`.

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

The handwritten `app/src/api/client.ts` wrapper supplies the API URL, timeouts (5 seconds; 15 seconds for creating a game and changing the route, which look up landmarks within a matching server budget: 3 s per Places call, 6 s per playground edge, 8 s per request, 12 s write timeout), readable errors, and basic runtime response validation. Its types and HTTP request functions come from generated code, not manually duplicated interfaces or URLs. Creation is not automatically retried: if the network drops after a save, an unreferenced session may remain in the database.

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

1. Change the Go response/request structs, handler, and its entry in the operation table in `backend/api/spec.go`.
2. From `backend/`, run `go generate ./cmd/server`, then from `app/` run `npm run api:generate` (requires local Go and Node).
3. Run `npm run typecheck`, `npm run lint`, `npm run test:api`, and the Go tests.
4. Include both regenerated outputs with your changes.

| Command | What it does |
| --- | --- |
| `go generate ./cmd/server` (in `backend/`) | Generate the API document from Go |
| `npm run api:generate` (in `app/`) | Regenerate the frontend client from the document |

Frontend tests check JSON requests, bearer tokens, successful responses, errors with HTTP status, runtime guards, timeouts, the server-clock countdown, and grid geometry/labels. Backend tests cover the generated API document's routes, domain parsing, lobby rules (countdown start/cancel, joining limits, actions after start), isolated service behavior, real SQLite persistence including position history and its cleanup, HTTP create and lobby flows, authentication, invalid requests, storage errors, and grid geometry. Contract checks are smoke tests, **not full OpenAPI schema validation**.

Docker builds do not run tests or regenerate anything; CI (`.github/workflows/ci.yml`) runs the tests and fails on stale generated files. Update the generated files before rebuilding after an API change.

## Run with Docker (recommended)

Install and start **Docker Desktop** with Linux containers and Docker Compose v2. You do not need Go or Node installed on your computer for this workflow.

From the repository root:

```sh
docker compose up --build
```

- **API:** <http://localhost:8080/health>

The backend image compiles a static Go executable into a minimal image without a shell (it runs as root). Compose has no health check; it starts Expo once the backend container is running.

SQLite is stored at `/data/catch-me.sqlite` in the `game_data` Docker volume. Container restarts/recreation and `docker compose down` preserve saved games. **Do not use `docker compose down -v` unless you intend to delete all volume data.** For a plain file copy, stop the backend first to avoid an inconsistent copy; the [SQLite snapshot workflow below](#inspect-a-local-database-snapshot) also works while the backend is running. Direct `docker run` users must also mount a persistent volume at `/data`.

The app container is an **Expo development server**, not a production frontend or an Android/iOS emulator. It uses Node 24 and mounts source files for live reload without sharing Windows `node_modules` with Linux. The native app still runs in Expo Go or a development build on your device. Rebuild the app image after changing dependencies or other files not mounted in `compose.yaml`.

`tsconfig.json` stays writable inside the app image because Expo updates it during startup; it is deliberately not mounted as an individual file. Rebuild after changing the host's TypeScript configuration.

### Use a physical phone with Docker

Copy the **root** `.env.example` to `.env` (PowerShell):

```powershell
Copy-Item .env.example .env
```

Replace the values with your computer's LAN IP, for example:

```dotenv
EXPO_HOST=192.168.1.100
EXPO_PUBLIC_API_URL=http://192.168.1.100:8080
```

Restart with `docker compose up --build`. Open `exp://192.168.1.100:8081` in Expo Go, or scan the QR code shown in the Expo logs. Your phone and computer must share a network, and ports **8080 and 8081** must be allowed through the firewall on a trusted private network.

`EXPO_HOST` overrides the hostname Expo advertises, so the phone gets the computer's IP rather than the container's private IP. For an Android Studio emulator, use `EXPO_HOST=10.0.2.2` and `EXPO_PUBLIC_API_URL=http://10.0.2.2:8080`.

Some Expo Go environments require Expo CLI and Expo Go to be signed in to the same Expo account. Sign in inside the container with `docker compose exec app npx expo login`; its CLI state is kept in the `expo_state` volume. Do not bake credentials into images.

Compose takes its configuration from the **root `.env`** and passes the API URL to Expo. `app/.env.local` is for the non-Docker workflow only; dotenv loading is disabled inside the container to avoid conflicting settings.

### Inspect a local database snapshot

SQLite has no database server, network port, username, or password. To inspect data or try SQL changes in IntelliJ IDEA (with database support), DataGrip, or DB Browser for SQLite, open a **local snapshot**, not the live Docker volume.

After the backend has initialized the database and the app image has been built (`docker compose up -d --build`), run this in **PowerShell from the repository root**:

```powershell
$dbDir = Join-Path $env:TEMP ("catch-me-db-" + (Get-Date -Format "yyyyMMdd-HHmmss"))
New-Item -ItemType Directory -Path $dbDir | Out-Null

docker run --rm --network none --read-only `
  --mount type=volume,source=catch-me_game_data,target=/data,readonly `
  --mount "type=bind,source=$dbDir,target=/export" `
  --entrypoint node catch-me-app `
  -e "const {DatabaseSync,backup}=require('node:sqlite'); const db=new DatabaseSync('/data/catch-me.sqlite',{readOnly:true}); backup(db,'/export/catch-me.sqlite').then(()=>db.close()).catch(e=>{console.error(e);process.exitCode=1});"

if ($LASTEXITCODE -ne 0) { throw 'SQLite snapshot failed.' }
Write-Output "$dbDir\catch-me.sqlite"
```

This uses Node's SQLite backup API in the existing app image to create a consistent snapshot while the backend can keep running. The source volume is mounted read-only; only the new local snapshot is written. No local Node or SQLite installation is needed. The names `catch-me_game_data` and `catch-me-app` assume the default Compose project name (`catch-me`); adjust them if you override it.

In IntelliJ IDEA / DataGrip:

1. Open **Database → + → Data Source → SQLite**.
2. Select the printed `catch-me.sqlite` file (download the SQLite driver if prompted).
3. Open a query console and run SQL, for example:

```sql
SELECT name, team, ready, position_recorded_at
FROM players
ORDER BY joined_at DESC;
```

In DB Browser for SQLite, use **Open Database** and select the same file.

- The snapshot does **not** update automatically. Repeat the export and open the new file to inspect newer data.
- SQL changes to the snapshot do **not** affect the running game. Do not copy an edited snapshot back over the live database.
- Snapshots can contain sensitive player, location, and game-access data. Keep them out of Git and public shares; delete local copies when no longer needed. Live database retention does not remove exported copies.

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
go run ./cmd/server -addr :8080
```

This plain-HTTP starter is for trusted local development; use HTTPS and add authentication before deploying gameplay or location data.

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

On iPhone, Expo Go is enough: iOS uses Apple Maps, which needs no API key. Android Google Maps with our own key requires a native build configured as described below. Pan/zoom the map, tap to choose a center, and press **Create game**. The app shows the runner and hunter join codes; enter your name and join a team yourself. Other players enter a code and their name on the home screen. In the lobby, runners allow location access and walk to the route's start; everyone then taps **Ready**. The game starts 10 seconds after the last player is ready if both teams have players.

iOS simulator development requires macOS; a native iPhone build can be produced using macOS or EAS, then connect to the development server running on Windows.

## Test without moving (development only)

Set `TEST_SKIP_START_POSITION_CHECK=true` in the root `.env`, then run `docker compose up -d --force-recreate --wait backend` and reload Expo Go. Without Docker, run the Go server with `-test-skip-start-position-check=true`. It defaults to `false`.

This server-side test override bypasses the runner's position requirements for both Ready and the finish endpoint (`POST /api/me/finish`), including missing or stale reports, and prevents movement from automatically canceling readiness/the countdown. It never bypasses authentication, role, or game-phase checks. **Real GPS tracking and position updates remain unchanged**; no coordinates are simulated, and finishing still requires an explicit request. After an accepted finish, further position reports are rejected and the app stops sharing when it receives the finished runner status. The server reports the override in `startPositionCheckDisabled`; the app shows a banner naming both disabled checks in the lobby and game screen, enables Ready independently of location, and offers the finish icon to active runners regardless of location. Location permission is still needed to share your real position, but not to start or finish a test run.

The override affects all games on that development server while enabled, is not persisted in SQLite, and must not be used for real games or production. Set it back to `false` and restart/recreate the backend after testing. Any old simulated positions are replaced by the next real report. The normal start-cell rules remain the default.

## Maps setup

`react-native-maps` was installed using `npx expo install react-native-maps`, selecting the SDK 58-compatible version. Android uses `PROVIDER_GOOGLE`; iOS uses the default Apple Maps provider, which works in Expo Go and needs no API key. The grid is drawn with polygons and polylines. Manual selection requests no GPS permission and does not track the device.

For Android Google Maps with our own key, use a native build:

1. Create a Google Cloud project, configure billing, and enable **Maps SDK for Android**. Check current Google pricing and configure budget alerts.
2. Create a key restricted to package `com.catchme.app` plus the build's signing SHA-1, and to **Maps SDK for Android**. This is an initial app identifier; choose your final unique identifier before publishing and update key restrictions accordingly.
3. Set `GOOGLE_MAPS_KEY` in `app/.env.local` for local work, or in the root `.env` for Docker Expo. `app/app.config.js` passes it to the maps config plugin. Never commit real keys. Native SDK keys are embedded in binaries and must be protected with application/API restrictions, not treated as secret server credentials.
4. Build locally with `npx expo run:android`. This generates the native project from Expo config; do not hand-edit generated native directories. Alternatively configure EAS and supply the key through its build environment (EAS credentials/project setup is not included in this milestone).
5. Rebuild the native binary after changing keys or native configuration. Metro reloads and JavaScript exports do not update native configuration.

Use the appropriate signing SHA-1 for each Android build (development and Play signing certificates can differ). Missing/restricted keys can produce a blank map; they cannot be validated by TypeScript tests or CI bundle exports. Use an HTTPS backend if the native build blocks local HTTP; certificate verification must remain enabled.

See the [SDK 58 maps setup documentation](https://docs.expo.dev/versions/v58.0.0/sdk/map-view/) for provider and signing-key instructions.

## GitHub Actions

`.github/workflows/ci.yml` runs a single **Project checks** job on pull requests, pushes to `main`, and manual runs from GitHub's Actions tab.

It checks Go formatting, backend tests and vet, generated API freshness, frontend TypeScript and lint, API client/generation tests, and Android/iOS JavaScript bundle exports. Checks do not regenerate committed files; stale generated output fails CI and must be regenerated (`go generate`, `npm run api:generate`) before pushing.

The workflow uses Node 24 and Go 1.26, caches npm downloads and Go modules/builds, and requires no custom secrets. Actions are pinned to commit hashes, the repository token is read-only, and superseded runs are canceled.

This is validation only: it does not deploy containers, publish images, produce signed native app binaries, or submit to app stores. GitHub-hosted execution starts after the workflow is committed and pushed to GitHub. To block merges on failures, configure a branch protection/ruleset requiring **Project checks** after its first run.

## Checks

Run in `app/`:

```sh
npm run typecheck
npm run lint
npm run test:api
npx expo-doctor
```

Run in `backend/`:

```sh
go test ./...
go vet ./...
gofmt -l .
```

Use `npx expo install <package>` when adding app packages, so Expo selects compatible versions. Commit `app/package-lock.json` for reproducible installs. Do not use `npm audit fix --force` blindly: it can replace Expo with an incompatible SDK.
