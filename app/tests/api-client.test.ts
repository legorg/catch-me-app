import assert from 'node:assert/strict';
import { test } from 'node:test';

import {
  API_URL, ApiError, activateJoker, addSketch, checkBackend, clearSketches, createGame, getGameState, joinGame, leaveGame, setHunterTeam, setReady, updatePosition,
} from '../src/api/client';
import { isGameStateHunterResponse } from '../src/api/types';
import { previewPlayground } from '../src/playground';

const center = { latitude: 52.52, longitude: 13.405 };
const route = {
  start: { title: 'Siegessäule', coordinates: { latitude: 52.4741, longitude: 13.3302 } },
  stop: { title: 'Brandenburger Tor', coordinates: { latitude: 52.5163, longitude: 13.3777 } },
  distanceMeters: 5873.4,
};
const created = {
  game: { sessionId: '00112233445566778899aabbccddeeff', createdAt: '2026-11-08T12:00:00Z', playground: previewPlayground(center), route },
  joinCodes: { runner: '482913', hunter: '071564' },
};
const token = 'ab'.repeat(32);
const state = {
  sessionId: created.game.sessionId,
  status: 'lobby',
  serverTime: '2026-11-08T12:00:00Z',
  playground: created.game.playground,
  playerId: '8899aabbccddeeff0011223344556677',
  team: 'runner',
  joinCodes: { runner: '482913', hunter: '071564' },
  players: [{ playerId: '8899aabbccddeeff0011223344556677', name: 'Ada', team: 'runner', ready: false }],
  route,
};
const { route: _route, ...withoutRoute } = state;
const hunterState = { ...withoutRoute, team: 'hunter', jokers: [], sketches: [] };

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
  assert.deepEqual(await activateJoker(token, 'showRunners'), used);
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    assert.deepEqual(await request.json(), { kind: 'queryZone', cell: 'C4' });
    return Response.json(used);
  });
  assert.deepEqual(await activateJoker(token, 'queryZone', 'C4'), used);
  t.mock.method(globalThis, 'fetch', async () => Response.json({ error: 'your hunter team already used this joker' }, { status: 409 }));
  await assert.rejects(activateJoker(token, 'showRunners'), /already used/);
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
