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
- Whenever the start condition stops holding during the countdown (someone joins, leaves, or un-readies), the countdown is cancelled. It restarts from 10 seconds once the condition holds again.
- Once the countdown has passed, the game is **running**. Joining, leaving, and changing ready state are then rejected.
- Players can leave during the lobby and the countdown.

Game status values: `lobby`, `countdown`, `running` (and `ended` from milestone 4).

## Runner routes [next: milestone 2]

- In the lobby, each runner gets a random **start cell in one edge row** (row 1 or row 6) and an **end cell in the opposite edge row**. Which edge is the start is random per runner.
- Each runner can **regenerate** their route **2 times**, only while not ready.
- Hunters have no assigned start point.
- A runner's route is visible to that runner only (teammates may see it from milestone 3 on, decided then).

## Positions and visibility [next: milestone 3]

- Foreground location only (app open). Players send their position regularly while the game is running.
- Only the **latest position** per player is stored. All positions are deleted when the game ends or the player leaves.
- **Teammates** always see each other's latest position.
- **Hunters** receive each active runner's position once every **5 minutes** (periodic reveal). The reveal shows the position at reveal time, not live updates.
- **Runners** never see hunters.
- Players outside the playground get a warning only; there is no penalty.

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

## API (milestone 1)

| Method and path | Auth | Purpose |
|---|---|---|
| `POST /api/games` | – | Create a game. Returns the game and both join codes. |
| `GET /api/games/{sessionId}` | – | Load the playground of a saved game (no codes, no players). |
| `POST /api/players` | – | Join with `{code, name}`. Returns the player token and the game state. |
| `GET /api/me/game` | Bearer | Game state as seen by the requesting player. |
| `PUT /api/me/ready` | Bearer | Set `{ready: true/false}`. Returns the game state. |
| `DELETE /api/me` | Bearer | Leave the game (lobby and countdown only). |

The game state contains the status, the server time, the scheduled start time during the countdown, the playground, the requesting player's ID and team, their team's join code, and the list of players. Clients compute the countdown from `startsAt - serverTime` to avoid depending on the phone's clock.

Errors: `400` invalid input, `401` missing or unknown token, `404` unknown game or join code, `409` rule violation (game started, full, name taken).

## Tuning values

| Value | Default |
|---|---|
| Countdown | 10 s |
| Max players per game | 30 |
| Name length | 1–20 characters |
| Route regenerations | 2 |
| Catch radius / position age | 50 m / 30 s |
| Reveal interval | 5 min |
| Time limit | 90 min |
