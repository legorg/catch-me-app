# Catch me! frontend end-to-end test specification

Status legend: **[done]** implemented, **[next]** agreed as the next step, **[planned]** agreed, details still open, **[open]** not decided yet.

## Goal

Automated end-to-end tests that drive the installed app through real screens against a real backend and run in the GitHub Actions pipeline. They complement, not replace, the existing backend Go tests and the app's logic tests (`npm run test:api`).

## Current pipeline [done]

- `.github/workflows/ci.yml` (every push/PR): `gofmt`, `go test ./...`, `go vet ./...`, generated API freshness, app `typecheck`, `lint`, `test:api`, and `expo export` for Android and iOS (bundle check only), plus DevGuard scans.
- `.github/workflows/release.yml` (tags): `typecheck`, `lint`, `test:api`, then an EAS production build.
- No UI or end-to-end tests run anywhere. `app/tests/*.test.ts` test pure logic only; no screen is rendered. `test:lint-rules` and `knip` are not part of either pipeline.

## Framework decision

- **Maestro [next]:** black-box YAML flows against an installed build, automatic waiting, recommended by Expo. Needs no edits to the generated native directories. GPS can be set (`setLocation`, `travel`), and `runScript` can call HTTP endpoints.
- Not chosen: **Detox** (gray-box and fast, but complex native setup and often late support for new React Native versions), **Appium** (heavy and slower), **Playwright/Cypress** on Expo Web (no `react-native-web`; maps, location and background tracking are native, so they would test a different app).
- **React Native Testing Library [open]:** possible later complement for component tests without a device; not end-to-end.

## Test environment

- **Platform [done]:** Android emulator on a Linux GitHub Actions runner. **iOS [open]:** needs macOS runners (much more expensive); decide later.
- **Runner choice:** GitHub Actions rather than EAS Workflows, because the backend must be reachable from the device. On a GitHub runner the Go API runs next to the emulator, reachable at `http://10.0.2.2:8080`. EAS Workflows would need a publicly hosted backend, which does not exist yet (see the game spec's "Deployment and pipeline").
- **Separate job:** APK build (prebuild + Gradle) plus emulator take roughly 15-25 minutes, so E2E runs as its own workflow (`.github/workflows/e2e.yml`) and not inside the 15-minute `checks` job. It runs on pushes to `main` and manually; **open:** also on pull requests.
- **Database:** a fresh SQLite file per run; never a shared or persistent database.

## Rules for tests

- **Real positions:** use Maestro's `setLocation`/`travel` to satisfy the start and finish position checks (50 m). Do not use `TEST_SKIP_START_POSITION_CHECK`; tracking and reporting stay real.
- **Several players [decided]:** one emulator plus API-driven players. The app under test controls one player. Other players (joining, Ready, position reports, catching) are driven through the backend API from the flow (`runScript`), using the generated API contract, not a separate fake game model. The app's role changes between flows (runner with API hunters, hunter with API runners, spectator), so every role's UI is tested against real opponents. Several emulators side by side were rejected for now: Maestro runs flows on several devices independently without synchronization points, two emulators strain the standard runner and add flakiness; reconsider only for single tests that need two real app UIs at once (e.g. teammates seeing each other).
- **Selectors:** add stable `testID`s to the elements the flows use. Do not rely on translated texts, which break with i18n changes.
- **Maps:** without a Google Maps key the map renders empty. Do not assert on map tiles. Avoid gestures such as turning the grid; use the default orientation.
- **Permissions:** grant location permissions at launch (`launchApp` permissions) instead of tapping system dialogs.
- **Privacy:** test data is synthetic; no real player positions in fixtures, logs or artifacts.

## Steps

Each step is delivered and reviewed on its own.

1. **Local Maestro setup [done]:** `app/.maestro/smoke.yml` (app starts, home screen visible) against a locally installed Android release build, the first `testID`s on the home screen (`home-join-code`, `home-join`, `home-create`) and README instructions. Passed locally on an API 37 emulator.
2. **Pipeline job [done, not yet run on GitHub]:** `.github/workflows/e2e.yml`, a separate job on pushes to `main` and manual runs: builds an x86_64 release APK, boots an API 35 emulator, runs the Maestro flows and uploads the JUnit report, screenshots and logs on failure. Starting the Go backend with a fresh database moves to step 3, because the smoke flow does not need it.
3. **Backend for flows [next]:**
   - **Places key [done]:** the server starts without a Google Places key; then only routes chosen on the map work, generated routes fail with 503 without calling Google. E2E flows therefore always choose the route on the map and need no secret.
   - **Cleartext HTTP [done]:** `E2E_BUILD=true` makes `app/app.config.js` set `usesCleartextTraffic` on Android, so the E2E release APK can reach `http://10.0.2.2:8080`; other builds keep blocking cleartext HTTP. The E2E workflow sets it.
   - **Backend in CI [done, not yet run on GitHub]:** the E2E workflow builds the Go server, starts it in the background on port 8080 with a fresh SQLite file in the runner's temp directory and without a Places key, waits up to 30 s for `/health`, and uploads `backend.log` with the Maestro results on failure.
   - **API players and first real flow [done, locally]:** `app/.maestro/scripts/create-game.js` (game with a route chosen on the map, join codes in `output.game`) and `join-player.js` (`CODE`, `NAME`, token in `output.players[NAME]`); `join-lobby.yml`: an API hunter joins, the app joins as a runner by code and name, and the lobby shows the app's player (`lobby-me`) and the API hunter. `subflows/wait-accepting-location-accuracy.yml` waits for an element and meanwhile accepts Google Play services' "Location Accuracy" dialog, which high-accuracy tracking shows once per device (by resource ID, not text). Swipes start below the map, since swiping on the map pans it. Passed locally three times in a row; **in CI it needs the Maps key** (the lobby shows a map).
   - **Android Maps key [done, not yet run on GitHub]:** the "Generate Android project" step pulls the EAS `production` environment with `EXPO_TOKEN`, extracts only `GOOGLE_MAPS_KEY` (readable there, not secret), deletes the file, masks the key, unsets `EXPO_TOKEN` and passes the key to `expo prebuild` alone; a missing or secret key fails the step with a clear message. **Needs:** the key's Android restriction must also allow `de.loewenburg.catchme` with the CI debug SHA-1 `5E:8F:16:06:2E:A3:CD:2C:4A:0D:54:78:76:BA:A6:F3:8C:AB:F6:25` (set in Google Cloud Console).
4. **Game flows [planned]:** extend step by step:
   - create game (playground, settings, codes),
   - join by code and name, hunter team selection,
   - lobby, Ready and countdown,
   - running game (positions, reveals, jokers),
   - catch and finish, spectator view,
   - automatic end and result screen.
5. **Store screenshots [planned]:** phone screenshots for Google Play from separate Maestro flows (`app/.maestro/store/`, tag `store`) with `takeScreenshot`, in a separate workflow (manual and on release tags) that always uploads them as an artifact; upload to the Play Console stays manual for now. Agreed: emulator profile 1080×1920 (9:16), Android demo mode for a clean status bar, German and English via the in-app language switch, synthetic game situations driven through the API, real map tiles (needs the Maps key). Before that, the home screen's development hint and server address are shown only in development builds (`__DEV__`). **Open:** which screens, automatic Play upload.
6. **iOS [open]:** same flows on an iOS simulator once cost and runner are decided.
