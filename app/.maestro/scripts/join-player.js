// Joins a player through the API (POST /api/players) with CODE and NAME, and stores
// the player's token in output.players[NAME] for later API calls on their behalf.
const api = typeof API_URL !== 'undefined' ? API_URL : 'http://127.0.0.1:8080';

const response = http.post(api + '/api/players', {
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ code: CODE, name: NAME }),
});
if (!response.ok) {
  throw new Error('join ' + NAME + ' failed: ' + response.status + ' ' + response.body);
}

output.players = output.players || {};
output.players[NAME] = json(response.body).playerToken;
