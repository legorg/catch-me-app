import assert from 'node:assert/strict';
import { test } from 'node:test';

import {
  API_URL, ApiError, checkBackend, createGame, getGameState, joinGame, leaveGame, regenerateRoute, setHunterTeam, setReady, updatePosition,
} from '../src/api/client';
import { previewPlayground } from '../src/playground';

const center = { latitude: 52.52, longitude: 13.405 };
const created = {
  game: { sessionId: '00112233445566778899aabbccddeeff', createdAt: '2026-11-08T12:00:00Z', playground: previewPlayground(center) },
  joinCodes: { runner: 'K7M2QX', hunter: 'P4R9TW' },
};
const token = 'ab'.repeat(32);
const state = {
  sessionId: created.game.sessionId,
  status: 'lobby',
  serverTime: '2026-11-08T12:00:00Z',
  playground: created.game.playground,
  playerId: '8899aabbccddeeff0011223344556677',
  team: 'runner',
  joinCode: 'K7M2QX',
  players: [{ playerId: '8899aabbccddeeff0011223344556677', name: 'Ada', team: 'runner', ready: false }],
  route: { start: 'B1', end: 'E6', regenerationsLeft: 2 },
};

test('create posts the center and cell size as JSON', async (t) => {
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    const url = new URL(request.url);
    assert.equal(url.origin, new URL(API_URL).origin);
    assert.equal(url.pathname, '/api/games');
    assert.equal(request.method, 'POST');
    assert.equal(request.headers.get('Content-Type'), 'application/json');
    assert.deepEqual(await request.json(), { ...center, cellSizeMeters: 2000 });
    return Response.json(created, { status: 201 });
  });
  assert.deepEqual(await createGame(center), created);
});

test('custom cell sizes are sent', async (t) => {
  for (const size of [100, 1500, 5000]) {
    const response = { ...created, game: { ...created.game, playground: previewPlayground(center, size) } };
    t.mock.method(globalThis, 'fetch', async (request: Request) => {
      assert.deepEqual(await request.json(), { ...center, cellSizeMeters: size });
      return Response.json(response, { status: 201 });
    });
    assert.deepEqual(await createGame(center, size), response);
  }
});

test('invalid input is rejected before a request', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => { assert.fail('should not fetch'); });
  for (const size of [0, 99, 1550, 5100, NaN, Infinity]) {
    await assert.rejects(createGame(center, size), /Cell size/);
  }
  for (const code of ['', 'ABC', 'ABCDE0', 'ABCDE1', 'ABCDEFG']) {
    await assert.rejects(joinGame(code, 'Ada'), /join code/);
  }
  for (const name of ['', '   ', 'x'.repeat(21)]) {
    await assert.rejects(joinGame('K7M2QX', name), /name/);
  }
});

test('join normalizes the code and trims the name', async (t) => {
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    assert.equal(new URL(request.url).pathname, '/api/players');
    assert.equal(request.method, 'POST');
    assert.equal(request.headers.get('Authorization'), null);
    assert.deepEqual(await request.json(), { code: 'K7M2QX', name: 'Ada' });
    return Response.json({ playerToken: token, state }, { status: 201 });
  });
  assert.deepEqual(await joinGame(' k7m2qx ', ' Ada '), { playerToken: token, state });
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
  assert.deepEqual(await regenerateRoute(token), state);
  assert.deepEqual(await setHunterTeam(token, ' Wolves '), state);
  assert.deepEqual(await updatePosition(token, { latitude: 52.5, longitude: 13.4, accuracyMeters: 8 }), state);
  await leaveGame(token);
  assert.deepEqual(seen, [
    'GET /api/me/game', 'PUT /api/me/ready', 'POST /api/me/route', 'PUT /api/me/hunter-team', 'PUT /api/me/position', 'DELETE /api/me',
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

test('API errors keep the message and status', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => Response.json({ error: 'missing or invalid player token' }, { status: 401 }));
  await assert.rejects(getGameState(token), (err: unknown) =>
    err instanceof ApiError && err.status === 401 && err.message === 'missing or invalid player token');
  t.mock.method(globalThis, 'fetch', async () => Response.json({ error: 'name is already taken in this game' }, { status: 409 }));
  await assert.rejects(joinGame('K7M2QX', 'Ada'), /name is already taken/);
});

test('malformed successful responses are rejected at runtime', async (t) => {
  for (const body of [null, {}, { ...created, joinCodes: { runner: 'bad', hunter: 'P4R9TW' } },
    { ...created, game: { ...created.game, sessionId: 'bad' } },
    { ...created, game: { ...created.game, playground: { ...created.game.playground, rows: 8 } } },
    { ...created, game: { ...created.game, playground: { ...created.game.playground, bounds: { ...created.game.playground.bounds, north: 0 } } } },
  ]) {
    t.mock.method(globalThis, 'fetch', async () => Response.json(body, { status: 201 }));
    await assert.rejects(createGame(center), /Unexpected game response/);
  }
  for (const body of [{ ...state, status: 'ended' }, { ...state, team: 'gamemaster' }, { ...state, joinCode: 'bad' },
    { ...state, startsAt: 'bad' }, { ...state, players: [{ ...state.players[0], ready: 'yes' }] }, { ...state, players: null },
    { ...state, route: { start: 'G1', end: 'E6', regenerationsLeft: 2 } }, { ...state, route: { ...state.route, regenerationsLeft: 3 } },
    { ...state, route: { ...state.route, end: null } }, { ...state, team: 'hunter', joinCode: 'P4R9TW' },
    { ...state, nextRevealAt: 'bad' }, { ...state, startPositionCheckDisabled: 'true' },
    { ...state, players: [{ ...state.players[0], position: { latitude: 95, longitude: 0, accuracyMeters: 5, recordedAt: state.serverTime } }] },
    { ...state, players: [{ ...state.players[0], position: { latitude: 0, longitude: 0, accuracyMeters: -1, recordedAt: state.serverTime } }] },
    { ...state, players: [{ ...state.players[0], route: { start: 'Z9', end: 'A6', regenerationsLeft: 0 } }] }]) {
    t.mock.method(globalThis, 'fetch', async () => Response.json(body));
    await assert.rejects(getGameState(token), /Unexpected game state/);
  }
  t.mock.method(globalThis, 'fetch', async () => Response.json({ playerToken: 'short', state }, { status: 201 }));
  await assert.rejects(joinGame('K7M2QX', 'Ada'), /Unexpected join response/);
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
