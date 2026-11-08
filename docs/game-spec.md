# Catch me! game specification

Status legend: **[done]** implemented, **[next]** planned in the given milestone.

## Principles

- **No human gamemaster.** The backend is the referee. The person who creates a game picks the playground and then joins like every other player, with no extra rights.
- **Objective rules only.** Every rule must be decidable by the server from stored data: positions, timestamps, and player actions.
- **Server-enforced visibility.** Clients only receive what the requesting player may see. Hidden data never leaves the backend; the UI does not merely hide it.
- **Lazy time evaluation.** There is no game loop. Time-based transitions (countdown finished, reveal due, time limit reached) are derived from stored timestamps whenever a request touches the game. This survives restarts and needs no scheduler.
- **Friendly play.** GPS spoofing cannot be fully prevented; the rules aim to be fair among honest players and robust against normal GPS error.

## Roles

| Team | Goal |
|---|---|
| Runners | Reach their assigned end cell without being caught. |
| Hunters | Catch every runner before one of them finishes, or until time runs out. |

## Playground [done]

- North-aligned 6 × 6 grid, columns A–F (west to east), rows 1–6 (north to south).
- Cell side 100–5000 m in 100 m steps, default 2000 m.
- Center latitude within ±80°, no date-line crossing.

## Joining [done: milestone 1]

- Creating a game returns two **join codes**: one for runners, one for hunters. Codes are 6 characters from `ABCDEFGHJKLMNPQRSTUVWXYZ23456789` (no easily confused 0/O, 1/I). Input is case-insensitive.
- A player joins with a code and a **display name** (1–20 characters after trimming, no control characters, unique per game ignoring case). The code decides the team.
- Joining returns a secret **player token**. Every later request authenticates with `Authorization: Bearer <token>`. The server stores only a SHA-256 hash of the token.
- At most **30 players** per game.
- Joining is possible in the lobby and during the countdown, not after the game has started.
- In the lobby, each player sees all players (names, teams, ready state) and the join code of their **own** team, to invite teammates.

## Lobby and start [done: milestone 1]

- Each player toggles **Ready**.
- **Start condition:** every player is ready, and each team has at least one player.
- When the start condition becomes true, the server schedules the start **10 seconds** later (countdown).
- Whenever the start condition stops holding during the countdown (someone joins, leaves, un-readies, or a ready runner leaves their start cell), the countdown is cancelled. It restarts from 10 seconds once the condition holds again.
- Once the countdown has passed, the game is **running**. Joining, leaving, and changing ready state are then rejected.
- Players can leave during the lobby and the countdown.

Game status values: `lobby`, `countdown`, `running` (and `ended` from milestone 4).

## Runner routes [done: milestone 2]

- When a runner joins, they get a random **start cell in one edge row** (row 1 or row 6) and an **end cell in the opposite edge row**. Which edge is the start is random per runner; columns are random.
- Each runner can **regenerate** their route **2 times**, only before the start and while not ready. A regenerated route always differs from the previous one.
- Hunters have no assigned start point.
- **Runners see all runners' routes** (their own and their teammates'). Hunters never see routes.
- Runners who joined before routes existed have no route and skip the start cell check.

## Start cell [done: milestone 3]

- A runner can only get **ready** when their latest position lies **inside their start cell** and is at most **30 seconds** old.
- Before the start, a ready runner who reports a position outside their start cell becomes **not ready** automatically (which cancels a running countdown).
- Once running, runners move freely.

## Positions and visibility [done: milestone 3]

- Foreground location only, while the lobby/game screen is open. Runners share from joining (for the start cell check); everyone shares while running. Hunters do not share before the start.
- The app sends the latest fix every **5 seconds**, even when standing still. The server stamps each report with its own time and never trusts the phone's clock.
- Only the **latest position** per player is stored, never a history. A player's position is deleted when they leave; deleting all positions at the end of a game follows in milestone 4.
- Everyone always sees **their own** position.
- **Teammates** see each other's latest position **only while running**.
- **Hunters** receive each runner's position once every **5 minutes** after the start (first reveal 5 minutes after the start). A reveal shows where the runner was at the reveal time (their last report before it), not live updates. The state includes `nextRevealAt` for both teams.
- **Runners** never see hunters.
- Players outside the playground get a warning only; there is no penalty.
- Reveals are recorded lazily: the first request touching the game after a reveal time snapshots the runners' positions, and position reports always snapshot before storing the new position.

## Catch, finish, and end [next: milestone 4]

- **Catch:** a runner taps "I was caught". The server accepts it only if some hunter's last known position is within **50 m** of the runner and at most **30 seconds** old. An accepted catch puts the runner **out** (spectator). Otherwise the app shows "no hunter nearby".
- **Finish:** a runner whose position lies inside their end cell finishes.
- **Runners win** as soon as one runner finishes.
- **Hunters win** when every runner is caught, or when the **90-minute** time limit runs out.
- The game becomes `ended` with a winner; positions are deleted.

## Later

- **Jokers** (special one-off abilities) on top of periodic reveals.
- Real-time updates (server-sent events or WebSockets) instead of polling.
- Background location tracking.
- Production readiness: HTTPS, rate limiting, versioned migrations, cleanup of old games, privacy policy, store builds.

## API (milestones 1 to 3)

| Method and path | Auth | Purpose |
|---|---|---|
| `POST /api/games` | – | Create a game. Returns the game and both join codes. |
| `GET /api/games/{sessionId}` | – | Load the playground of a saved game (no codes, no players). |
| `POST /api/players` | – | Join with `{code, name}`. Returns the player token and the game state. |
| `GET /api/me/game` | Bearer | Game state as seen by the requesting player. |
| `PUT /api/me/ready` | Bearer | Set `{ready: true/false}`. Returns the game state. |
| `POST /api/me/route` | Bearer | Regenerate the runner's route. Returns the game state. |
| `PUT /api/me/position` | Bearer | Report `{latitude, longitude, accuracyMeters}`. Returns the game state. |
| `DELETE /api/me` | Bearer | Leave the game (lobby and countdown only). |

The game state contains the status, the server time, the scheduled start time during the countdown, the next reveal time while running, the playground, the requesting player's ID and team, their team's join code, their own route (runners), and the list of players. Each player entry carries only what the requester may see: routes (for runners) and a position (own, teammates while running, or a revealed runner position with `revealedAt` for hunters). Clients compute countdowns from the server time to avoid depending on the phone's clock.

Errors: `400` invalid input, `401` missing or unknown token, `404` unknown game or join code, `409` rule violation (game started, full, name taken, route change not allowed, runner not in start cell).

## Tuning values

| Value | Default |
|---|---|
| Countdown | 10 s |
| Max players per game | 30 |
| Name length | 1–20 characters |
| Route regenerations | 2 |
| Catch radius / position age | 50 m / 30 s |
| Reveal interval | 5 min |
| Start cell position max age | 30 s |
| Position send interval (app) | 5 s |
| Time limit | 90 min |
