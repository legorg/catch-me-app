import assert from 'node:assert/strict';
import { test } from 'node:test';

import {
  API_URL, ApiError, activateJoker, addSketch, checkBackend, clearSketches, createGame, finish, getGameResult, getGameState, getTimeline, joinGame, leaveGame, markCaught, regenerateRoute, setHunterTeam, setReady, updatePosition,
} from '../src/api/client';
import { isGameStateHunterResponse } from '../src/api/types';
import { previewPlayground } from '../src/playground';
import { isValidGameId, normalizeGameId } from '../src/game-result';

const center = { latitude: 52.52, longitude: 13.405 };
const jokers = {
  showRunners: { allowance: 1 }, queryZone: { allowance: 1 }, compass: { allowance: 1 },
  trap: { allowance: 1 }, zoneOnly: { allowance: 1 }, fakePosition: { allowance: 1 },
  showHunters: { allowance: 1 },
};
const route = {
  start: { title: 'Siegessäule', coordinates: { latitude: 52.4741, longitude: 13.3302 } },
  stop: { title: 'Brandenburger Tor', coordinates: { latitude: 52.5163, longitude: 13.3777 } },
  distanceMeters: 5873.4,
};
const created = {
  game: { sessionId: '00112233445566778899aabbccddeeff', createdAt: '2026-11-08T12:00:00Z', playground: previewPlayground(center), revealIntervalMinutes: 30, route },
  joinCodes: { runner: '482913', hunter: '071564', spectator: '305821' },
};
const token = 'ab'.repeat(32);
const state = {
  sessionId: created.game.sessionId,
  status: 'lobby',
  serverTime: '2026-11-08T12:00:00Z',
  playground: created.game.playground,
  playerId: '8899aabbccddeeff0011223344556677',
  team: 'runner',
  joinCodes: { runner: '482913', hunter: '071564', spectator: '305821' },
  players: [{ playerId: '8899aabbccddeeff0011223344556677', name: 'Ada', team: 'runner', ready: false, runner: { status: 'active' } }],
  route,
  jokers: [],
};
const { route: _route, ...withoutRoute } = state;
const hunterState = { ...withoutRoute, team: 'hunter', jokers: [], sketches: [] };

test('the spectator code joins by the same name-and-code flow and preserves the server spectator role', async (t) => {
  const spectator = {
    ...withoutRoute, route, role: 'spectator', team: 'spectator',
    players: [{ playerId: state.playerId, name: 'Viewer', team: 'spectator', ready: false }],
  };
  const requests: string[] = [];
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    requests.push(`${request.method} ${new URL(request.url).pathname}`);
    if (request.method === 'POST') {
      assert.deepEqual(await request.json(), { code: created.joinCodes.spectator, name: 'Viewer' });
      return Response.json({ playerToken: token, state: spectator });
    }
    assert.equal(request.headers.get('Authorization'), `Bearer ${token}`);
    return Response.json(spectator);
  });
  const joined = await joinGame(created.joinCodes.spectator, ' Viewer ');
  assert.deepEqual(joined.state, spectator);
  assert.equal(joined.playerToken, token);
  assert.deepEqual(await getGameState(token), spectator);
  assert.deepEqual(requests, ['POST /api/players', 'GET /api/me/game']);
});

const trackPoint = { latitude: 52.52, longitude: 13.405, accuracyMeters: 5, recordedAt: '2026-11-18T12:00:05Z' };
const gameResult = {
  sessionId: created.game.sessionId,
  startsAt: '2026-11-18T12:00:00Z', endedAt: '2026-11-18T13:00:00Z', durationSeconds: 3600,
  playground: created.game.playground, route,
  players: [
    { playerId: 'a'.repeat(32), name: 'Ada', team: 'runner', runner: { status: 'finished', endedAt: '2026-11-18T12:30:00Z' }, track: [trackPoint] },
    { playerId: 'b'.repeat(32), name: 'Cy', team: 'runner', runner: { status: 'caught', endedAt: '2026-11-18T13:00:00Z' }, track: [] },
    { playerId: 'c'.repeat(32), name: 'Bob', team: 'hunter', hunterTeam: 'Pack', track: [trackPoint, { ...trackPoint, recordedAt: '2026-11-18T12:00:10Z' }] },
  ],
  events: [
    { kind: 'gameStarted', at: '2026-11-18T12:00:00Z' },
    { kind: 'hunterJokerUsed', at: '2026-11-18T12:10:00Z', hunterTeam: 'Pack', joker: 'compass' },
    { kind: 'runnerFinished', at: '2026-11-18T12:30:00Z', runner: { playerId: 'a'.repeat(32), name: 'Ada' } },
    { kind: 'runnerCaught', at: '2026-11-18T13:00:00Z', runner: { playerId: 'b'.repeat(32), name: 'Cy' } },
    { kind: 'gameEnded', at: '2026-11-18T13:00:00Z' },
  ],
};

test('result lookup uses the normalized full game ID without a token, body or position', async (t) => {
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    assert.equal(request.method, 'GET');
    assert.equal(new URL(request.url).pathname, `/api/games/${gameResult.sessionId}/result`);
    assert.equal(request.headers.get('Authorization'), null);
    assert.equal(await request.text(), '');
    return Response.json(gameResult);
  });
  assert.deepEqual(await getGameResult(` ${gameResult.sessionId.toUpperCase()} `), gameResult);
});

test('invalid game IDs and join codes are rejected without making a result request', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => { assert.fail('must not fetch'); });
  assert.equal(normalizeGameId(` ${gameResult.sessionId.toUpperCase()} `), gameResult.sessionId);
  assert.equal(isValidGameId(gameResult.sessionId), true);
  for (const id of ['', '482913', 'a'.repeat(31), 'a'.repeat(33), 'z'.repeat(32), '../' + gameResult.sessionId]) {
    assert.equal(isValidGameId(id), false);
    await assert.rejects(getGameResult(id), /32 hexadecimal characters/);
  }
});

test('unknown or unfinished results retain 404, and a later retry can succeed', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => Response.json({ error: 'game not found' }, { status: 404 }));
  await assert.rejects(getGameResult(gameResult.sessionId), (err: unknown) => err instanceof ApiError && err.status === 404);
  t.mock.method(globalThis, 'fetch', async () => Response.json(gameResult));
  assert.deepEqual(await getGameResult(gameResult.sessionId), gameResult);
});

test('a result for another game is rejected', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => Response.json({ ...gameResult, sessionId: 'f'.repeat(32) }));
  await assert.rejects(getGameResult(gameResult.sessionId), /invalid game result/);
});

const ada = { playerId: 'a'.repeat(32), name: 'Ada' };
const timeline = {
  serverTime: '2026-11-18T13:00:10Z',
  events: [
    { kind: 'gameStarted', at: '2026-11-18T12:00:10Z' },
    { kind: 'hunterJokerUsed', at: '2026-11-18T12:07:00Z', hunterTeam: 'Wolves', joker: 'showRunners' },
    { kind: 'runnerJokerUsed', at: '2026-11-18T12:08:00Z', runner: ada, joker: 'zoneOnly' },
    { kind: 'runnerCaught', at: '2026-11-18T12:30:00Z', runner: ada },
    { kind: 'runnerFinished', at: '2026-11-18T13:00:10Z', runner: { playerId: 'b'.repeat(32), name: 'Cy' } },
    { kind: 'gameEnded', at: '2026-11-18T13:00:10Z' },
  ],
};

test('the timeline is fetched with the token and returned as sent', async (t) => {
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    assert.equal(request.method, 'GET');
    assert.equal(new URL(request.url).pathname, '/api/me/timeline');
    assert.equal(request.headers.get('Authorization'), `Bearer ${token}`);
    return Response.json(timeline);
  });
  assert.deepEqual(await getTimeline(token), timeline);
});

test('active players keep the 409 of the timeline request', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => Response.json({ error: 'only spectators can see the timeline' }, { status: 409 }));
  await assert.rejects(getTimeline(token), (err: unknown) => err instanceof ApiError && err.status === 409);
});

test('create posts the center and all settings as JSON, with defaults', async (t) => {
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    const url = new URL(request.url);
    assert.equal(url.origin, new URL(API_URL).origin);
    assert.equal(url.pathname, '/api/games');
    assert.equal(request.method, 'POST');
    assert.equal(request.headers.get('Content-Type'), 'application/json');
    assert.deepEqual(await request.json(), { ...center, cellSizeMeters: 2000, gridSize: 6, revealIntervalMinutes: 30, jokers });
    return Response.json(created, { status: 201 });
  });
  assert.deepEqual(await createGame(center), created);
});

test('custom settings are sent', async (t) => {
  for (const [size, cells, minutes] of [[100, 3, 1], [1500, 6, 30], [5000, 12, 60]]) {
    const response = { ...created, game: { ...created.game, playground: previewPlayground(center, size, cells), revealIntervalMinutes: minutes } };
    t.mock.method(globalThis, 'fetch', async (request: Request) => {
      assert.deepEqual(await request.json(), { ...center, cellSizeMeters: size, gridSize: cells, revealIntervalMinutes: minutes, jokers });
      return Response.json(response, { status: 201 });
    });
    assert.deepEqual(await createGame(center, size, cells, minutes), response);
  }
});

test('invalid input is rejected before a request', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => { assert.fail('should not fetch'); });
  for (const size of [0, 99, 1550, 5100, NaN, Infinity]) {
    await assert.rejects(createGame(center, size), /Cell size/);
  }
  for (const cells of [2, 13, 6.5, NaN]) {
    await assert.rejects(createGame(center, 2000, cells), /grid/);
  }
  for (const minutes of [0, 61, 0.5, NaN]) {
    await assert.rejects(createGame(center, 2000, 6, minutes), /reveal interval/);
  }
  for (const code of ['', '123', '12345A', 'K7M2QX', '1234567', '123 45']) {
    await assert.rejects(joinGame(code, 'Ada'), /join code/);
  }
  for (const name of ['', '   ', 'x'.repeat(21)]) {
    await assert.rejects(joinGame('482913', name), /name/);
  }
});

test('join normalizes the code and trims the name', async (t) => {
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    assert.equal(new URL(request.url).pathname, '/api/players');
    assert.equal(request.method, 'POST');
    assert.equal(request.headers.get('Authorization'), null);
    assert.deepEqual(await request.json(), { code: '482913', name: 'Ada' });
    return Response.json({ playerToken: token, state }, { status: 201 });
  });
  assert.deepEqual(await joinGame(' 482913 ', ' Ada '), { playerToken: token, state });
});

test('player requests send the bearer token', async (t) => {
  const seen: string[] = [];
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    assert.equal(request.headers.get('Authorization'), `Bearer ${token}`);
    seen.push(`${request.method} ${new URL(request.url).pathname}`);
    if (request.method === 'PUT' && new URL(request.url).pathname === '/api/me/ready') {
      assert.deepEqual(await request.json(), { ready: true });
    }
    if (request.method === 'PUT' && new URL(request.url).pathname === '/api/me/position') {
      assert.deepEqual(await request.json(), { latitude: 52.5, longitude: 13.4, accuracyMeters: 8 });
    }
    if (request.method === 'PUT' && new URL(request.url).pathname === '/api/me/hunter-team') {
      assert.deepEqual(await request.json(), { name: 'Wolves' });
    }
    if (request.method === 'DELETE') {
      return new Response(null, { status: 204 });
    }
    return Response.json(state);
  });
  assert.deepEqual(await getGameState(token), state);
  assert.deepEqual(await setReady(token, true), state);
  assert.deepEqual(await setHunterTeam(token, ' Wolves '), state);
  assert.deepEqual(await updatePosition(token, { latitude: 52.5, longitude: 13.4, accuracyMeters: 8 }), state);
  await leaveGame(token);
  assert.deepEqual(seen, [
    'GET /api/me/game', 'PUT /api/me/ready', 'PUT /api/me/hunter-team', 'PUT /api/me/position', 'DELETE /api/me',
  ]);
  await assert.rejects(setHunterTeam(token, '  '), /team name/);
});

test('health check also uses the generated client', async (t) => {
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    assert.equal(new URL(request.url).pathname, '/health');
    return Response.json({ status: 'ok', service: 'catch-me-backend' });
  });
  assert.deepEqual(await checkBackend(), { status: 'ok', service: 'catch-me-backend' });
});

test('marking caught posts without coordinates and preserves the returned catch on retries', async (t) => {
  const caught = {
    ...state,
    status: 'running',
    players: [{ ...state.players[0], runner: { status: 'caught', endedAt: '2026-11-18T12:07:00Z', caughtPosition: { ...center, accuracyMeters: 8, recordedAt: '2026-11-18T12:06:58Z' } } }],
  };
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    assert.equal(request.method, 'POST');
    assert.equal(new URL(request.url).pathname, '/api/me/caught');
    assert.equal(request.headers.get('Authorization'), `Bearer ${token}`);
    assert.equal(await request.text(), '');
    return Response.json(caught);
  });
  assert.deepEqual(await markCaught(token), caught);
  assert.deepEqual(await markCaught(token), caught);
  t.mock.method(globalThis, 'fetch', async () => Response.json({ error: 'you can only mark yourself caught while the game is running' }, { status: 409 }));
  await assert.rejects(markCaught(token), (err: unknown) => err instanceof ApiError && err.status === 409);
});

test('finishing posts only the token and preserves the server outcome on retries', async (t) => {
  const finished = {
    ...state, status: 'running',
    players: [{ ...state.players[0], runner: { status: 'finished', endedAt: '2026-11-18T12:07:00Z' } }],
  };
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    assert.equal(request.method, 'POST');
    assert.equal(new URL(request.url).pathname, '/api/me/finish');
    assert.equal(request.headers.get('Authorization'), `Bearer ${token}`);
    assert.equal(await request.text(), '');
    return Response.json(finished);
  });
  assert.deepEqual(await finish(token), finished);
  assert.deepEqual(await finish(token), finished);
});

test('finish failures keep their HTTP status or network error, never a local success', async (t) => {
  for (const status of [401, 409]) {
    t.mock.method(globalThis, 'fetch', async () => Response.json({ error: 'finish refused' }, { status }));
    await assert.rejects(finish(token), (err: unknown) => err instanceof ApiError && err.status === status && err.message === 'finish refused');
  }
  t.mock.method(globalThis, 'fetch', async () => { throw new TypeError('Network request failed'); });
  await assert.rejects(finish(token), /Network request failed/);
});

test('the game state is returned as sent', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => Response.json(hunterState));
  assert.deepEqual(await getGameState(token), hunterState);
});

test('API errors keep the message and status', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => Response.json({ error: 'missing or invalid player token' }, { status: 401 }));
  await assert.rejects(getGameState(token), (err: unknown) =>
    err instanceof ApiError && err.status === 401 && err.message === 'missing or invalid player token');
  t.mock.method(globalThis, 'fetch', async () => Response.json({ error: 'name is already taken in this game' }, { status: 409 }));
  await assert.rejects(joinGame('482913', 'Ada'), /name is already taken/);
});

test('using a joker posts its kind with the token and returns the state', async (t) => {
  const used = {
    ...hunterState,
    jokers: [{
      kind: 'showRunners', available: false, usedAt: '2026-11-08T12:07:00Z',
      positions: [{ playerId: state.playerId, latitude: 52.5, longitude: 13.4, accuracyMeters: 5, recordedAt: '2026-11-08T12:06:58Z' }],
    }, { kind: 'queryZone', available: false, usedAt: '2026-11-08T12:08:00Z', cell: 'C4', found: false }],
  };
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    assert.equal(new URL(request.url).pathname, '/api/me/jokers');
    assert.equal(request.method, 'POST');
    assert.equal(request.headers.get('Authorization'), `Bearer ${token}`);
    assert.deepEqual(await request.json(), { kind: 'showRunners' });
    return Response.json(used);
  });
  assert.deepEqual(await activateJoker(token, { kind: 'showRunners' }), used);
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    assert.deepEqual(await request.json(), { kind: 'queryZone', cell: 'C4' });
    return Response.json(used);
  });
  assert.deepEqual(await activateJoker(token, { kind: 'queryZone', cell: 'C4' }), used);
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    assert.deepEqual(await request.json(), { kind: 'fakePosition', latitude: 52.5, longitude: 13.4 });
    return Response.json(used);
  });
  assert.deepEqual(await activateJoker(token, { kind: 'fakePosition', latitude: 52.5, longitude: 13.4 }), used);
  t.mock.method(globalThis, 'fetch', async () => Response.json({ error: 'your hunter team already used this joker' }, { status: 409 }));
  await assert.rejects(activateJoker(token, { kind: 'showRunners' }), /already used/);
});

test('sketches are posted and cleared with the token', async (t) => {
  const sketch = { sketchId: '0123456789abcdef0123456789abcdef', createdAt: '2026-11-08T12:07:00Z', points: [center, { latitude: 52.53, longitude: 13.41 }] };
  const runningState = { ...hunterState, status: 'running', sketches: [sketch] };
  const seen: string[] = [];
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    assert.equal(request.headers.get('Authorization'), `Bearer ${token}`);
    seen.push(`${request.method} ${new URL(request.url).pathname}`);
    if (request.method === 'POST') {
      assert.deepEqual(await request.json(), { points: sketch.points });
      return Response.json(runningState);
    }
    return Response.json({ ...runningState, sketches: [] });
  });
  // Extra fields of a point are not sent.
  assert.deepEqual(await addSketch(token, sketch.points.map((p) => ({ ...p, extra: 1 }))), runningState);
  const cleared = await clearSketches(token);
  assert.ok(isGameStateHunterResponse(cleared) && cleared.sketches.length === 0);
  assert.deepEqual(seen, ['POST /api/me/sketches', 'DELETE /api/me/sketches']);

  for (const points of [[], [center], Array(501).fill(center)]) {
    await assert.rejects(addSketch(token, points), /sketch needs/);
  }
});

test('network failures are propagated', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => { throw new TypeError('Network request failed'); });
  await assert.rejects(createGame(center), /Network request failed/);
});

test('requests time out and abort the fetch', async (t) => {
  t.mock.method(globalThis, 'setTimeout', (callback: () => void, delay: number) => {
    assert.equal(delay, 5000);
    queueMicrotask(callback);
    return 0;
  });
  t.mock.method(globalThis, 'clearTimeout', () => {});
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    assert.equal(request.signal.aborted, true);
    throw new Error('Aborted');
  });
  await assert.rejects(getGameState(token), /Connection timed out/);
});

test('landmark lookups wait longer and explain a timeout', async (t) => {
  const delays: number[] = [];
  t.mock.method(globalThis, 'setTimeout', (callback: () => void, delay: number) => {
    delays.push(delay);
    queueMicrotask(callback);
    return 0;
  });
  t.mock.method(globalThis, 'clearTimeout', () => {});
  t.mock.method(globalThis, 'fetch', async () => { throw new Error('Aborted'); });
  await assert.rejects(createGame(center), /Finding landmarks for the route took too long/);
  await assert.rejects(regenerateRoute(token), /Finding a new route took too long/);
  assert.deepEqual(delays, [15_000, 15_000]);
});
