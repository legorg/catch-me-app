# Catch me! game specification

Status legend: **[done]** implemented, **[next]** planned in the given milestone, **[planned]** agreed, rules still open.

## Principles

- **No human gamemaster.** The backend is the referee. The person who creates a game picks the playground and then joins like every other player, with no extra rights.
- **Objective rules only.** Every rule must be decidable by the server from stored data: positions, timestamps, and player actions.
- **Server-enforced visibility.** Clients only receive what the requesting player may see. Hidden data never leaves the backend; the UI does not merely hide it.
- **Lazy time evaluation.** There is no game loop. Time-based transitions (countdown finished, reveal due, time limit reached) are derived from stored timestamps whenever a request touches the game. This survives restarts and needs no scheduler.
- **Friendly play.** GPS spoofing cannot be fully prevented; the rules aim to be fair among honest players and robust against normal GPS error.

## Roles

| Team | Goal |
|---|---|
| Runners | Reach the route's stop landmark without being caught. |
| Hunters | Catch every runner before one of them finishes, or until time runs out. |

## Playground [done]

- North-aligned square grid of 3 × 3 to 12 × 12 cells, chosen when creating the game, default 6 × 6. Columns A, B, … (west to east), rows 1, 2, … (north to south), e.g. A1 to L12 for 12 × 12.
- Cell side 100–5000 m in 100 m steps, default 2000 m.
- Center latitude within ±80°, no date-line crossing.
- The **reveal interval** (see Visibility) is chosen when creating the game: 1–60 whole minutes, default 30.

## Joining [done: milestone 1]

- Creating a game returns two **join codes**: one for runners, one for hunters. Codes are 6 digits (`000000` to `999999`, leading zeros count) so they can be typed on a number pad.
- A player joins with a code and a **display name** (1–20 characters after trimming, no control characters, unique per game ignoring case). The code decides the team.
- Joining returns a secret **player token**. Every later request authenticates with `Authorization: Bearer <token>`. The server stores only a SHA-256 hash of the token.
- At most **30 players** per game.
- Joining is possible in the lobby and during the countdown, not after the game has started.
- In the lobby, each player sees all players (names, teams, hunter teams, ready state) and **both** join codes, so anyone (including the creator after joining) can invite players to either team.

## Hunter teams [done]

- Runners are always one team. Hunters can form several **hunter teams**; all hunters share the same hunter join code.
- A hunter joins without a team. In the lobby and during the countdown, a hunter who is not ready **chooses a hunter team by name**: an existing team with that name (ignoring case, keeping its spelling) is joined, otherwise a new one is created. Names follow the display name rules (1–20 characters).
- A hunter team exists exactly while at least one hunter belongs to it; a team left by its last member disappears. There is no separate team record.
- A hunter can only get ready with a hunter team, so every hunter team has a player and every hunter has a team when the game starts. The development override does not skip this.
- Hunters win together (all hunter teams).

## Lobby and start [done: milestone 1]

- Each player toggles **Ready**.
- **Background location [app only]:** in installed builds, the app enables Ready only once "Always" location access is granted, so positions keep coming while the screen is off during the game. The server cannot verify the permission and does not check it; it can also be withdrawn later. Expo Go cannot share in the background, so there the app only requires foreground location access.
- **Start condition:** every player is ready, and each team has at least one player (hunters: see hunter teams).
- When the start condition becomes true, the server schedules the start **10 seconds** later (countdown).
- Whenever the start condition stops holding during the countdown (someone joins, leaves, un-readies, or a ready runner leaves the start), the countdown is cancelled. It restarts from 10 seconds once the condition holds again.
- Once the countdown has passed, the game is **running**. Joining, leaving, and changing ready state are then rejected.
- Players can leave during the lobby and the countdown.

Game status values: `lobby`, `countdown`, `running` (and `ended` from milestone 4).

## Route [done]

- Each game has **one route** that all runners share: a **start** landmark in a random cell of the southern edge row (the last row) and a **stop** landmark in a random cell of the northern edge row (row 1). The server picks both from a landmark lookup when the game is created; creating fails if an edge has no landmark.
- **Any runner** can **regenerate** the route, **2 times per game** in total (shared by all runners). Only in the lobby (not during the countdown or once running) and only while **no runner is ready**.
- Regenerating looks up the start and the stop again, each avoiding the current landmark. An edge without another landmark keeps its current one; if neither edge has another landmark, regenerating is refused and the count stays.
- Hunters have no start point. **Runners see the route**; hunters never see it.

## Start position [done: milestone 3]

- A runner can only get **ready** when their latest position lies within **50 m** of the route's start landmark and is at most **30 seconds** old.
- Before the start, a ready runner who reports a position farther away becomes **not ready** automatically (which cancels a running countdown).
- Once running, runners move freely.

Development testing only: `TEST_SKIP_START_POSITION_CHECK=true` (server flag `-test-skip-start-position-check=true`) temporarily bypasses this Ready/countdown position gate. Real position reporting is unchanged. The override defaults to off, applies to the development server rather than a stored game rule, and is exposed as `startPositionCheckDisabled` with a test banner in the app. Never use it for real games.

## Positions and visibility [done: milestone 3]

- Before the start, location is shared in the foreground only, while the lobby screen is open. Everyone shares from joining: runners for the start position check, hunters so their hunter team sees them.
- **Background sharing [done]:** while the game runs, players who grant "Always" location access keep reporting with the screen off or another app open (Android shows a notification while sharing). It stops when the player leaves the game screen, closes the app, or the token becomes invalid; a restarted app does not resume it. It needs an installed app build; in Expo Go and browsers the app falls back to foreground sharing.
- The app sends the latest fix every **5 seconds**, even when standing still. The server stamps each report with its own time and never trusts the phone's clock.
- Every reported position is stored as its own record; the newest is the player's current position. A player's positions are deleted when they leave.
- **Position history [stored only]:** all records together are kept for a replay or chart after the game. No endpoint or app screen reads them yet; once one exists, it must only show them after the game ended. Positions are deleted automatically **7 days after the game's start** (or creation, if it never started).
- Everyone always sees **their own** position.
- **Hunters of the same hunter team** see each other's latest position **at any time**, including the lobby. Hunters never see other hunter teams' positions.
- **Runners** see each other's latest position **only while running**.
- **Hunters** receive each runner's position once every **reveal interval** after the start (the game's setting, 1–60 minutes, default 30; first reveal one interval after the start). A reveal shows where the runner was at the reveal time (their last report before it), not live updates. The state includes `nextRevealAt` and `revealIntervalMinutes` for both teams. Hunters get **every** reveal so far, oldest first (`reveals` per runner), not just the latest; the app fades older ones.
- **Runners** never see hunters, except the snapshot of the show hunters joker (see Jokers).
- Players outside the playground get a warning only; there is no penalty.
- Reveals are not stored: each reveal is derived from the stored positions as the runner's last report strictly before that reveal time. A reveal time before the runner's first report has no reveal.

## Catch, finish, and end [next: milestone 4]

- **Catch:** a runner taps "I was caught". The server accepts it only if some hunter's last known position is within **50 m** of the runner and at most **30 seconds** old. An accepted catch puts the runner **out** (spectator). Otherwise the app shows "no hunter nearby".
- **Finish:** a runner who is not out finishes when they report a position within **50 m** of the route's stop landmark (the same radius as the start check).
- **Runners win** as soon as one runner finishes.
- **Hunters win** when every runner is caught, or when the **90-minute** time limit runs out.
- The game becomes `ended` with a winner; positions stay stored until their retention ends, for the replay.

## Next steps

Agreed topics whose rules are still open. Each needs its rules settled here before it is implemented.

### Deployment and pipeline [planned]

- Decide **where** the backend runs (hosting, domain, HTTPS) and **how** the app is distributed (EAS builds, internal testing, stores).
- Extend the GitHub Actions pipeline (`.github/workflows/ci.yml`) from checks to build and deploy: backend image, app builds, configuration and secrets per environment.
- Production readiness still needed before real games: HTTPS, rate limiting, versioned migrations instead of AutoMigrate, cleanup of old games, privacy policy, an API versioning and compatibility policy (see API).
- **Remove development logging** before production: the `landmark lookup` lines (duration and cells per playground edge) exist only to tune the landmark time budget; drop or downgrade them to debug once the budget is settled. Also review the remaining logs, e.g. the route coordinates in `route created`/`route regenerated` and the app's `[api]` request log and `console.log` of errors in `app/src/api/client.ts`, and keep only what operations need.
- Open: hosting target, environments (staging/production), who triggers a deployment.

### Game result and history view [planned, after milestone 4]

- After the game ends, show the result (winner, catches, finish) to all players.
- Show the stored position history as a replay or chart of everyone's movements. It is only available after the game ended (see position history).
- Open: what exactly the result shows, replay vs. static track chart, how long it stays reachable (positions are deleted 7 days after the start).

### Jokers [hunter jokers implemented, runner jokers implemented]

One-off special abilities on top of the periodic reveals. A joker belongs to a team: runners, or one hunter team (hunters of the same team share it). The app has a joker sheet (the "Jokers" button on the game screen) listing the team's jokers.

**Show runners** (hunter teams) [implemented]: once per game and hunter team, while the game is running, any hunter of the team can use it. The team then sees where every runner was at that moment (each runner's last report at or before the use). The result stays on the map, in a different color than the reveals, and is visible only to that hunter team; other hunter teams still have their own joker, and runners are not told. The server stores only that the team used it (and when); the positions are derived from the stored position history, like reveals. Using it again is a `409`.

Every joker can be used once per game and team, only while the game is running; using it again is a `409`. The server enforces all of them.

**Query a zone** (hunter teams) [implemented]: any hunter of the team picks one cell of the game's grid (in the app: a cell picker in the joker sheet; a cell outside the grid is a `400`). The team gets a yes/no answer: was at least one runner inside that cell at that moment (each runner's last report at or before the use; a runner without a report counts as outside)? The answer does not say how many runners or which. The cell and the answer stay on the team's map; only that hunter team sees them, and runners are not told. The server stores the cell and the time; the answer is derived from the position history.

**Compass** (hunter teams) [implemented]: any hunter of the team can use it, but only after reporting a position (otherwise a `409`, and the joker stays available). The team learns in which **direction** every runner was at that moment, seen from that hunter's last reported position at the use (the **origin**): for each runner, the bearing from the origin to the runner's last report at or before the use, in whole degrees clockwise from true north (0–359). A runner without a report is left out. The server never sends the runners' positions or distances, and the bearings carry no player IDs; they are sorted by angle, so they do not reveal which runner is which. The app draws one arrow of fixed length (one cell width) per bearing, starting at the origin, without names. The result stays on the team's map; only that hunter team sees it, and runners are not told. The server stores the origin and the time; the bearings are derived from the position history, like reveals.

**Runner jokers** belong to the runner team as a whole: each exists once per game, and when one runner uses it, it is used up for all runners. All runners see which runner team jokers were used, by whom and when. Hunters are never told that a joker was used. The reveal jokers (show only the zone, fake position) affect only the runner who used it, and only that runner's **next reveal**; hunters only see their effect at the reveal. A runner can have at most one pending reveal joker at a time (a second one before the next reveal is a `409`).

- **Show hunters** [implemented]: any runner can use it. All runners then see where every hunter (of every hunter team) was at that moment (each hunter's last report at or before the use; a hunter without a report is left out). It is a snapshot, not live: the result stays on the runners' map in its own color. Hunters are not told. It acts immediately and does not touch any reveal, so the pending-joker rule does not apply: it neither blocks nor is blocked by a pending reveal joker. The server stores who used it and when; the positions are derived from the stored position history, like reveals.

- **Show only the zone** [implemented]: at the next reveal (the first reveal time strictly after the use), hunters see only the cell this runner is in **at that reveal** (their last report before it, like every reveal), not the exact position; the app highlights the cell on the map with the reveal's numbered marker in its middle. If the runner is outside the playground then, the reveal has neither cell nor position (no marker, but the numbering keeps the gap). The server stores who used it and when; the zone is derived like reveals, and the position of that reveal is never sent to hunters. Runners see the joker in their joker sheet, and once used, who used it and when.
- **Fake position** [implemented]: the runner picks any point inside the playground (in the app by moving a map until a crosshair is on it; a point outside is a `400`). At the next reveal (the first reveal time strictly after the use), hunters see that point instead of the real position, looking like a normal reveal: it keeps the accuracy and report time of the real position, only the coordinates are replaced. If the runner has no report before that reveal, there is no reveal, as usual. The server stores who used it, when, and the point; all runners see the point (as a marker on their map). The server never sends the real position of that reveal to hunters (before the game has ended); the faked reveal stays faked.

### Hunter team sketches [done: MVP]

- Hunters of one hunter team draw freehand lines (**sketches**) on the map that only their team sees. Runners and other hunter teams never receive them; the server filters them like positions.
- Drawing is possible **only while the game is running**, for hunters in a hunter team, on the game screen's map ("Draw" button; the map stops moving while drawing). A line is sent when the finger is lifted; teammates see it with the next game state poll (no live drawing). The app thins longer lines to 500 points.
- A sketch is a list of 2–500 points (latitude ±80°, longitude ±180°), in one color in the MVP. A hunter team has at most 200 sketches.
- **Any team member can clear all** of the team's sketches at once. There is no undo or deleting single lines in the MVP.
- Sketches belong to the hunter team, not to the drawing player: they stay when a player leaves. (Leaving is not possible while running, and hunter teams only change before the start, so a team cannot disappear while it has sketches.)
- Sketches are kept for the later replay and deleted **7 days after the game's start**, like positions.
- Later: colors, undo, deleting single lines, notes or markers, sketches for runners.

### Share link for a game [planned]

- Invite players with a link instead of typing a code: opening it starts the app (or the store/browser) with the join code filled in, so only the name is missing.
- Open: one link per team or one link with a team choice, link format and domain (deep links need the deployment's domain and app configuration), what happens without the app installed, sharing via the system share sheet.

### Moving the playground square [planned]

- When creating a game, the creator can drag the grid on the map to fine-tune its position, instead of only placing it by choosing the center.
- Open: dragging the whole square vs. moving the map under a fixed square, whether it also works with the Apple Maps and Google Maps views, whether the position can still change after creation (probably not once players joined, since routes depend on the grid).

### Further ideas

- Real-time updates (server-sent events or WebSockets) instead of polling.
- Background location that survives an app restart (needs a persisted player token).
- Translated backend error messages (currently English; the app prevents most of them by validating input first).

## API (milestones 1 to 3)

**Compatibility:** the game is not live yet, so breaking API changes are allowed; the app and the backend are always changed and shipped together (the generated client comes from the same commit). There is no API versioning: paths are unversioned (`/api/...`), there is no version header, and the OpenAPI document's version stays at `0.1.0`. Before the first release, decide on a versioning and compatibility policy (installed apps then lag behind the backend).

| Method and path | Auth | Purpose |
|---|---|---|
| `POST /api/games` | – | Create a game `{latitude, longitude, cellSizeMeters?, gridSize?, revealIntervalMinutes?}` (omitted settings get their defaults). Returns the game (with `revealIntervalMinutes`) and both join codes. |
| `POST /api/players` | – | Join with `{code, name}`. Returns the player token and the game state. |
| `GET /api/me/game` | Bearer | Game state as seen by the requesting player. |
| `PUT /api/me/ready` | Bearer | Set `{ready: true/false}`. Returns the game state. |
| `POST /api/me/route` | Bearer | Regenerate the game's route (runners, see Route). Returns the game state. |
| `PUT /api/me/hunter-team` | Bearer | Join or create the hunter team `{name}` (hunters, not ready, before the start). Returns the game state. |
| `PUT /api/me/position` | Bearer | Report `{latitude, longitude, accuracyMeters}`. Returns the game state. |
| `POST /api/me/jokers` | Bearer | Use a joker of the player's team: hunters in a hunter team `{kind: "showRunners"}`, `{kind: "queryZone", cell: "C4"}`, or `{kind: "compass"}`, runners `{kind: "zoneOnly"}`, `{kind: "fakePosition", latitude, longitude}`, or `{kind: "showHunters"}` (exactly the fields of that joker, anything else is a `400`; while running, once each per team; a runner whose last reveal joker has not acted yet must wait for their next reveal before using another reveal joker). Returns the game state. |
| `POST /api/me/sketches` | Bearer | Add a sketch `{points: [{latitude, longitude}, …]}` for the player's hunter team (while running; body up to 64 KiB, other endpoints 1024 bytes). Returns the game state. |
| `DELETE /api/me/sketches` | Bearer | Clear all sketches of the player's hunter team (while running). Returns the game state. |
| `DELETE /api/me` | Bearer | Leave the game (lobby and countdown only). |

The game state contains the status, the server time, the scheduled start time during the countdown, the next reveal time while running, the playground, the requesting player's ID and team, both join codes, the game's route and `routeRegenerationsLeft` (runners only), and the list of players. Each player entry carries only what the requester may see: the hunter team (for hunters who chose one), the `jokers` of the requester's own team (hunters in a hunter team, and all runners), each a variant picked by its `kind` with `available` and, once used, `usedAt` plus only that joker's result (`positions` for showRunners, `cell` and `found` for queryZone, `origin` and `bearings` for compass, `usedBy` for zoneOnly, `usedBy` and `point` for fakePosition, `usedBy` and `positions` for showHunters), a live position (own, hunter teammates always, runner teammates while running) and, for hunters looking at a runner while running, the list of `reveals`, each a variant picked by its `kind`: `position` (with `revealedAt` and the `position`), `zone` (with `revealedAt` and only the `cell`), or `outside` (only `revealedAt`). Runners never get reveals, and hunters never get a live runner position. While running, hunters in a hunter team also get their team's `sketches` (oldest first, each with `id`, `createdAt`, and `points`); everyone else gets an empty list. Clients compute countdowns from the server time to avoid depending on the phone's clock.

Errors: `400` invalid input, `401` missing or unknown token, `404` unknown game or join code, `409` rule violation (game started, full, name taken, route regeneration not allowed or no other route available, hunter team change not allowed, runner not at the start, hunter without a team, joker not available: game not running, already used, not a joker of the player's team, a hunter without a team, or the runner's last reveal joker has not acted yet; a compass before the hunter reported a position; sketch not allowed: game not running, not a hunter in a team, or the team's sketch limit reached).

## Tuning values

| Value | Default |
|---|---|
| Countdown | 10 s |
| Max players per game | 30 |
| Name length | 1–20 characters |
| Route regenerations | 2 per game |
| Start radius | 50 m |
| Finish radius | 50 m |
| Catch radius / position age | 50 m / 30 s |
| Reveal interval | 1–60 min per game, default 30 min |
| Start position max age | 30 s |
| Position send interval (app) | 5 s |
| Time limit | 90 min |
| Points per sketch | 2–500 |
| Sketches per hunter team | 200 |
