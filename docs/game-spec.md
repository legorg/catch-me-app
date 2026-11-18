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
| Hunters | Catch runners before they reach the route's stop landmark. Final team scoring is still open (see Finish and end). |

## Playground [done]

- North-aligned square grid of 3 × 3 to 12 × 12 cells, chosen when creating the game, default 6 × 6. Columns A, B, … (west to east), rows 1, 2, … (north to south), e.g. A1 to L12 for 12 × 12.
- Cell side 100–5000 m in 100 m steps, default 2000 m.
- Center latitude within ±80°, no date-line crossing.
- The **reveal interval** (see Visibility) is chosen when creating the game: 1–60 whole minutes, default 30.
- **Joker allowances [planned]:** when creating the game, choose an allowance of **0–5 uses for each joker kind** (inclusive; 0 disables it). See Jokers for ownership and open configuration details.

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
- **Leaving while running [planned]:** players can also leave an active game. A runner who leaves no longer counts as active for the end condition below. Closing the app or losing connectivity alone does not count as leaving. The rules above describe the current implementation until this is added.

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
- Active players always see **their own** position. A caught runner instead gets the frozen catch location described below.
- **Hunters of the same hunter team** see each other's latest position **at any time**, including the lobby. Hunters never see other hunter teams' positions.
- **Runners** see each other's latest position **only while running**.
- **Hunters** receive each runner's position once every **reveal interval** after the start (the game's setting, 1–60 minutes, default 30; first reveal one interval after the start). A reveal shows where the runner was at the reveal time (their last report before it), not live updates. The state includes `nextRevealAt` and `revealIntervalMinutes` for both teams. Hunters get **every** reveal so far, oldest first (`reveals` per runner), not just the latest; the app fades older ones.
- **Runners** never see hunters, except the snapshot of the show hunters joker (see Jokers).
- Players outside the playground get a warning only; there is no penalty.
- Reveals are not stored: each reveal is derived from the stored positions as the runner's last report strictly before that reveal time. A reveal time before the runner's first report has no reveal.
- **Caught runners [backend done; app next]:** everyone immediately receives their `caughtAt` and, if a report exists, `caughtPosition` (the last real report before declaring the catch). Their normal live position and all periodic reveals are omitted, including for themselves. Further reports are rejected. The catch location uses the existing position history, so its coordinates disappear with normal retention; the caught status remains. New hunter location jokers exclude caught runners. Earlier joker snapshots remain historical results; the app must suppress their normal markers for caught runners when drawing the catch marker.
- **Finished runners [planned]:** after an accepted finish, stop foreground/background location reporting and reject further reports on the server. Hunters must not receive the runner's finish coordinates or the route's stop landmark, including through later periodic reveals, location jokers, traps, or notifications. In particular, do not introduce a finish-position marker analogous to the public catch marker. Previously permitted historical snapshots must not be updated to the finish position. The post-game result may expose stored tracks only once the entire game has ended.

## Catch [backend done; app next], finish and end [next: milestone 4]

- **Catch [backend implemented]:** while running, a runner can permanently mark **themselves** caught, on trust alone. No hunter proximity, position freshness, or reported position is required. The server timestamps the catch. Repeated requests succeed without changing the original catch time or location. Hunters and players in the lobby/countdown cannot declare a catch (`409`). There is no undo.
- A caught runner is **out**: no more position reports or joker uses (`409`), and no available jokers. Their token remains valid for watching with the same runner visibility rules; there is no privileged spectator view. Existing position history is preserved for the later replay and normal retention.
- **App [next]:** add an "I was caught" button with confirmation, stop foreground/background location reporting after a confirmed catch, and replace the runner's ordinary markers with a black "Caught" marker at `caughtPosition`, visible to everyone immediately (on the next state poll, not waiting for a reveal). Without a report, show only the caught status. The game screen stays accessible for watching.
- **No automatic end yet:** even if every runner is caught, the game remains `running`. Finishing, winner determination, and the time limit below remain unimplemented.
- **Finish [planned]:** while running, an active runner within **50 m** of the route's stop landmark sees a **"Ziel erreicht"** button. Entering the radius alone does not finish the runner: they must press the button. The backend validates the radius from the runner's reported position before marking them finished and recording the finish time. Position freshness for this validation is still to be decided.
- **After finishing [planned]:** the runner is no longer active; location reporting stops (see visibility above). Finishing is permanent. Retry behavior, remaining joker access, and the watching UI must be specified before implementation; finishing must not reveal the destination to hunters.
- **End condition [planned]:** the game ends automatically when **every runner has either been caught, finished, or left the game**. The first finish alone does **not** end the game while another runner is still active. The server records `ended` and the end time; all remaining gameplay location reporting stops. Stored positions remain subject to normal retention.
- **Scoring and time limit [open]:** define the winner/statistics for mixed outcomes (finished, caught, left). The former "first finisher immediately wins and ends the game" rule is superseded. The previously proposed **90-minute** time limit is not implemented; whether it remains as an additional end condition, and how unfinished runners are scored then, must be settled before implementation.

## Next steps

Agreed topics whose rules are still open. Each needs its rules settled here before it is implemented.

### Deployment and pipeline [planned]

- Decide **where** the backend runs (hosting, domain, HTTPS) and **how** the app is distributed (EAS builds, internal testing, stores).
- Extend the GitHub Actions pipeline (`.github/workflows/ci.yml`) from checks to build and deploy: backend image, app builds, configuration and secrets per environment.
- Production readiness still needed before real games: HTTPS, rate limiting, versioned migrations instead of AutoMigrate, cleanup of old games, privacy policy, an API versioning and compatibility policy (see API).
- **Remove development logging** before production: the `landmark lookup` lines (duration and cells per playground edge) exist only to tune the landmark time budget; drop or downgrade them to debug once the budget is settled. Also review the remaining logs, e.g. the route coordinates in `route created`/`route regenerated` and the app's `[api]` request log and `console.log` of errors in `app/src/api/client.ts`, and keep only what operations need.
- Open: hosting target, environments (staging/production), who triggers a deployment.

### Game result and history view [planned, after milestone 4]

- Add **"Spielergebnis ansehen"** to the frontend start screen. The user enters a **game ID** (not a runner or hunter join code) to open the result; joining the game is not required by this flow.
- The view is available **only for ended games**. The backend must enforce this, not just hide the screen: ongoing games must never expose result/history data through this lookup.
- Add a **new backend result endpoint addressed by game ID**, returning the result, statistics, and historical data needed for the graphics. Exact path, response schema, and access/privacy policy for a game-ID lookup must be agreed before implementation; the ID is not a player token, and no player tokens or token hashes may be returned.
- Show player outcomes (caught, finished, left), relevant times, game statistics, and historical graphics based on stored movement/event data. Include joker effects and other gameplay events needed to explain the match, distinguishing real tracks from fake reveals and other snapshots. The exact statistics and visualization (replay, static tracks, charts) remain to be specified.
- **Automatic navigation:** when an active game's state changes to `ended`, stop location reporting and automatically open this same result view for that game ID, rather than a separate result screen implementation.
- **Reproducible development test data:** provide a development/test-only way to generate an already completed game and its game ID, including position histories and use of **all implemented gameplay features**, extended as planned features are implemented. Cover catches, finishes, leaving, reveals, every joker kind (including zone alerts and trap placement/activation/trigger), hunter teams, and sketches so the result/statistics/graphics can be tested without playing an entire match. Use synthetic data, not real player locations; no production seed endpoint or simulated live GPS tracking. Fixture timestamps must allow the history to be inspected before retention removes it.
- **Retention [open]:** positions and sketches currently expire 7 days after the game's start, and leaving currently deletes a player's positions. Decide how the result handles expired/deleted tracks, how long results stay reachable, and whether leaving should instead retain history until normal expiry. Do not silently extend retention or promise a full replay after data has been deleted.

### Notifications [planned]

- Integrate app notifications that can appear on the **lock screen**, including while the app is in the background. Request the required OS permissions and make disabled/denied notifications visible in the app; actual display depends on device permissions and OS settings.
- At minimum, support the new runner zone alert and triggered-trap notification described below. Further notification events (e.g. game end) are still to be selected.
- Notifications obey the same server-side team/role visibility rules as the game state. Do not put hidden coordinates, the destination, secrets, or player tokens into lock-screen text or notification payloads. Deliver an event only to its intended recipients and avoid duplicate notifications for the same event.
- Delivery architecture (including push delivery when the app is not polling), device registration, and event persistence/retry rules are still to be designed. Notification delivery must not require replacing the backend's lazy game-rule evaluation with a game loop; evaluate location-triggered events when location reports arrive. Account for the existing limits of background location reporting.

### Jokers [existing kinds implemented; new kinds and configurable allowances planned]

One-off special abilities on top of the periodic reveals. A joker belongs to a team (runners, or one hunter team), except **fake position**, which belongs to each runner individually. The app has a joker sheet (the "Jokers" button on the game screen) listing the player's personal and team jokers.

**Show runners** (hunter teams) [implemented]: once per game and hunter team, while the game is running, any hunter of the team can use it. The team then sees where every runner was at that moment (each runner's last report at or before the use). The result stays on the map, in a different color than the reveals, and is visible only to that hunter team; other hunter teams still have their own joker, and runners are not told. The server stores only that the team used it (and when); the positions are derived from the stored position history, like reveals. Using it again is a `409`.

Every joker can be used once per game and team, except **fake position**, which can be used once per game **per runner**. Jokers can only be used while the game is running; using a spent joker again is a `409`. The server enforces all of them.

**Configurable allowances [planned]:** replace the fixed one-use limits above and below with a per-kind **0–5** allowance chosen at game creation. Preserve ownership: shared runner jokers spend the runner team's allowance, hunter jokers spend the respective hunter team's allowance, and personal jokers spend the individual runner's allowance. The backend stores/enforces the settings and remaining uses; the creation UI configures them and the joker UI shows remaining uses. Existing descriptions marked implemented still describe today's one-use behavior. Defaults, exact API fields, and how the automatic zone alert's per-opponent limit interacts with allowances greater than one remain open. A configured 0 must disable that kind. Multiple allowed uses must retain their individual results/history rather than overwrite an earlier use; the existing one-pending-reveal-joker-per-runner rule remains in force.

Caught runners cannot use any joker. Hunter location jokers used at or after a runner's catch exclude that runner; earlier snapshots keep their historical result when reloaded.

**Query a zone** (hunter teams) [implemented]: any hunter of the team picks one cell of the game's grid (in the app: a cell picker in the joker sheet; a cell outside the grid is a `400`). The team gets a yes/no answer: was at least one runner inside that cell at that moment (each runner's last report at or before the use; a runner without a report counts as outside)? The answer does not say how many runners or which. The cell and the answer stay on the team's map; only that hunter team sees them, and runners are not told. The server stores the cell and the time; the answer is derived from the position history.

**Compass** (hunter teams) [implemented]: any hunter of the team can use it, but only after reporting a position (otherwise a `409`, and the joker stays available). The team learns in which **direction** every runner was at that moment, seen from that hunter's last reported position at the use (the **origin**): for each runner, the bearing from the origin to the runner's last report at or before the use, in whole degrees clockwise from true north (0–359). A runner without a report is left out. The server never sends the runners' positions or distances, and the bearings carry no player IDs; they are sorted by angle, so they do not reveal which runner is which. The app draws one arrow of fixed length (one cell width) per bearing, starting at the origin, without names. The result stays on the team's map; only that hunter team sees it, and runners are not told. The server stores the origin and the time; the bearings are derived from the position history, like reveals.

**Runner jokers:** show hunters and show only the zone belong to the runner team as a whole: each exists once per game, and when one runner uses it, it is used up for all runners. All runners see which shared runner jokers were used, by whom and when. **Fake position is personal:** each runner has their own use, status, and chosen point; using it does not spend another runner's joker. The joker sheet and fake-position marker show only the requesting runner's own use. Hunters are never told that a joker was used. The reveal jokers (show only the zone, fake position) affect only the runner who used it, and only that runner's **next reveal**; hunters only see their effect at the reveal. A runner can have at most one pending reveal joker at a time (a second one before the next reveal is a `409`). Different runners can each have a fake-position joker pending at the same time.

- **Show hunters** [implemented]: any runner can use it. All runners then see where every hunter (of every hunter team) was at that moment (each hunter's last report at or before the use; a hunter without a report is left out). It is a snapshot, not live: the result stays on the runners' map in its own color. Hunters are not told. It acts immediately and does not touch any reveal, so the pending-joker rule does not apply: it neither blocks nor is blocked by a pending reveal joker. The server stores who used it and when; the positions are derived from the stored position history, like reveals.

- **Show only the zone** [implemented]: at the next reveal (the first reveal time strictly after the use), hunters see only the cell this runner is in **at that reveal** (their last report before it, like every reveal), not the exact position; the app highlights the cell on the map with the reveal's numbered marker in its middle. If the runner is outside the playground then, the reveal has neither cell nor position (no marker, but the numbering keeps the gap). The server stores who used it and when; the zone is derived like reveals, and the position of that reveal is never sent to hunters. Runners see the joker in their joker sheet, and once used, who used it and when.
- **Fake position** [implemented]: once per game per runner, the runner picks any point inside the playground (in the app by moving a map until a crosshair is on it; a point outside is a `400`). At the next reveal (the first reveal time strictly after the use), hunters see that point instead of the real position, looking like a normal reveal: it keeps the accuracy and report time of the real position, only the coordinates are replaced. If the runner has no report before that reveal, there is no reveal, as usual. The server stores who used it, when, and the point; the runner sees their own point as a marker on their map. The server never sends the real position of that reveal to hunters (before the game has ended); the faked reveal stays faked. Existing stored uses remain spent for the runner who used them; other runners retain their personal use.

- **Hunter-team zone alert** (runners) [planned]: automatically notify an active runner the **first time any hunter of a particular hunter team shares the runner's grid cell**. The baseline limit is **once per game, per hunter team, per runner**: another member of that hunter team or a later re-entry does not notify that runner again, but another hunter team can, and each runner has their own alert state. This grants no live hunter coordinates. Evaluate on real position reports from either side. Open: position freshness, whether the notification identifies the hunter team/cell, and whether the configurable allowance counts distinct hunter-team alerts or changes the once-per-team rule. No repeated-alert behavior is agreed yet.
- **Trap** (hunter teams) [planned]: a hunter selects a point on the map to place a trap, spending their hunter team's allowance. The trap becomes active **20 minutes after placement**, measured by server time; before then it cannot trigger. An active trap triggers when an active runner is within **100 m** of its point. **The affected runner team receives the notification** (not a hunter notification by default). Trap evaluation must not leak the runner's position or the destination to hunters. Open: valid placement area, whether one trigger consumes the trap, lifetime, visibility of trap markers, notification contents, multiple runners/traps triggering together, and whether activation may use an existing fresh report or must wait for a new report. Trap timing follows lazy evaluation; no notification at the exact activation instant without a qualifying request is promised.

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
| `POST /api/me/caught` | Bearer | Permanently mark the requesting runner caught, while running, with no request body or proximity check. Idempotent: retries preserve the original catch. Returns the game state. `409` for hunters or before the game is running. |
| `PUT /api/me/ready` | Bearer | Set `{ready: true/false}`. Returns the game state. |
| `POST /api/me/route` | Bearer | Regenerate the game's route (runners, see Route). Returns the game state. |
| `PUT /api/me/hunter-team` | Bearer | Join or create the hunter team `{name}` (hunters, not ready, before the start). Returns the game state. |
| `PUT /api/me/position` | Bearer | Report `{latitude, longitude, accuracyMeters}`. Returns the game state. Caught runners receive `409`; no report is stored. |
| `POST /api/me/jokers` | Bearer | Use a personal or team joker: hunters in a hunter team `{kind: "showRunners"}`, `{kind: "queryZone", cell: "C4"}`, or `{kind: "compass"}`, runners `{kind: "zoneOnly"}`, `{kind: "fakePosition", latitude, longitude}`, or `{kind: "showHunters"}` (exactly the fields of that joker, anything else is a `400`; while running, fakePosition once per runner, other jokers once each per team; a runner whose last reveal joker has not acted yet must wait for their next reveal before using another reveal joker). Returns the game state. |
| `POST /api/me/sketches` | Bearer | Add a sketch `{points: [{latitude, longitude}, …]}` for the player's hunter team (while running; body up to 64 KiB, other endpoints 1024 bytes). Returns the game state. |
| `DELETE /api/me/sketches` | Bearer | Clear all sketches of the player's hunter team (while running). Returns the game state. |
| `DELETE /api/me` | Bearer | Leave the game (lobby and countdown only). |

The game state contains the status, the server time, the scheduled start time during the countdown, the next reveal time while running, the playground, the requesting player's ID and team, both join codes, the game's route and `routeRegenerationsLeft` (runners only), and the list of players. Each player entry carries only what the requester may see: the hunter team (for hunters who chose one), the `jokers` of the requester's own team (hunters in a hunter team, and all runners), each a variant picked by its `kind` with `available` and, once used, `usedAt` plus only that joker's result (`positions` for showRunners, `cell` and `found` for queryZone, `origin` and `bearings` for compass, `usedBy` for zoneOnly, `usedBy` and `point` for fakePosition, `usedBy` and `positions` for showHunters), a live position (own, hunter teammates always, runner teammates while running) and, for hunters looking at a runner while running, the list of `reveals`, each a variant picked by its `kind`: `position` (with `revealedAt` and the `position`), `zone` (with `revealedAt` and only the `cell`), or `outside` (only `revealedAt`). Runners never get reveals, and hunters never get a live runner position. While running, hunters in a hunter team also get their team's `sketches` (oldest first, each with `id`, `createdAt`, and `points`); everyone else gets an empty list. Clients compute countdowns from the server time to avoid depending on the phone's clock.

Errors: `400` invalid input, `401` missing or unknown token, `404` unknown game or join code, `409` rule violation (game started, full, name taken, route regeneration not allowed or no other route available, hunter team change not allowed, runner not at the start, hunter without a team, joker not available: game not running, already used, not a joker of the player's team, a hunter without a team, or the runner's last reveal joker has not acted yet; a compass before the hunter reported a position; sketch not allowed: game not running, not a hunter in a team, or the team's sketch limit reached).

### Planned API extensions (not implemented)

- Explicit runner finish action with server-side 50 m validation; finished status/time in permitted game-state responses without exposing finish coordinates to hunters.
- Leaving a running game and automatic end evaluation when no active runner remains; expose the end time and game ID so the app can navigate to the result.
- A game-ID-addressed result endpoint for ended games only, providing statistics and historical graphics data; define its access policy, error responses, and retention behavior before implementation.
- Creation settings for each joker kind's 0–5 allowance; support multiple uses/results, automatic zone-alert state, trap placement/state, and remaining-use counts.
- Notification device registration/delivery as required by the chosen notification design. Keep all generated API outputs unchanged until the corresponding backend implementation is requested.

## Tuning values

| Value | Default |
|---|---|
| Countdown | 10 s |
| Max players per game | 30 |
| Name length | 1–20 characters |
| Route regenerations | 2 per game |
| Start radius | 50 m |
| Finish radius | 50 m |
| Catch proximity check | None (self-declared, fairness-based) |
| Reveal interval | 1–60 min per game, default 30 min |
| Start position max age | 30 s |
| Position send interval (app) | 5 s |
| Time limit | Previously proposed 90 min; not implemented, role in new end rules open |
| Joker allowance [planned] | 0–5 per kind at creation; defaults open |
| Runner zone alert [planned] | Once per game, hunter team, and runner; interaction with allowance open |
| Trap activation delay [planned] | 20 min |
| Trap trigger radius [planned] | 100 m |
| Points per sketch | 2–500 |
| Sketches per hunter team | 200 |
