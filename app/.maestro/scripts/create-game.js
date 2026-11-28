// Creates a game through the API (POST /api/games) and stores its join codes in output.game.
// Runs on the computer running Maestro, so the backend is at 127.0.0.1, not the emulator's 10.0.2.2.
// The route is chosen on the map, so the backend needs no Google Places key.
// Synthetic playground around Berlin's center: a 6x6 grid of 2000 m cells.
const api = typeof API_URL !== 'undefined' ? API_URL : 'http://127.0.0.1:8080';
const allowance = { allowance: 1 };

const response = http.post(api + '/api/games', {
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    latitude: 52.52,
    longitude: 13.405,
    cellSizeMeters: 2000,
    gridSize: 6,
    rotationDegrees: 0,
    revealIntervalMinutes: 30,
    jokers: {
      showRunners: allowance,
      queryZone: allowance,
      compass: allowance,
      trap: allowance,
      showHunters: allowance,
      zoneOnly: allowance,
      fakePosition: allowance,
      zoneAlert: { enabled: false },
    },
    // Start in the southern edge row, stop in the northern edge row.
    route: {
      start: { latitude: 52.475, longitude: 13.405 },
      stop: { latitude: 52.565, longitude: 13.405 },
    },
  }),
});
if (!response.ok) {
  throw new Error('create game failed: ' + response.status + ' ' + response.body);
}

const created = json(response.body);
output.game = {
  sessionId: created.game.sessionId,
  runnerCode: created.joinCodes.runner,
  hunterCode: created.joinCodes.hunter,
  spectatorCode: created.joinCodes.spectator,
};
