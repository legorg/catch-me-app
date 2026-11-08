import assert from 'node:assert/strict';
import { test } from 'node:test';

import { API_URL, checkBackend, createGame, loadGame } from '../src/api/client';
import { previewPlayground } from '../src/playground';

const center = { latitude: 52.52, longitude: 13.405 };
const game = {
  sessionId: '00112233445566778899aabbccddeeff',
  createdAt: '2026-11-08T12:00:00Z',
  playground: previewPlayground(center),
};

test('generated create client posts the center as JSON', async (t) => {
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    const url = new URL(request.url);
    assert.equal(url.origin, new URL(API_URL).origin);
    assert.equal(url.pathname, '/api/games');
    assert.equal(request.method, 'POST');
    assert.equal(request.headers.get('Content-Type'), 'application/json');
    assert.deepEqual(await request.json(), { ...center, cellSizeMeters: 2000 });
    return Response.json(game, { status: 201 });
  });
  assert.deepEqual(await createGame(center), game);
});

test('custom and legacy cell sizes are sent and accepted on load', async (t) => {
  for (const size of [100, 1500, 2000, 5000]) {
    const saved = { ...game, playground: previewPlayground(center, size) };
    t.mock.method(globalThis, 'fetch', async (request: Request) => {
      if (request.method === 'POST') assert.deepEqual(await request.json(), { ...center, cellSizeMeters: size });
      return Response.json(saved);
    });
    assert.deepEqual(await createGame(center, size), saved);
    assert.deepEqual(await loadGame(saved.sessionId), saved);
  }
});

test('invalid cell sizes are rejected before creating a game', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => { assert.fail('should not fetch'); });
  for (const size of [0, 99, 1550, 5100, NaN, Infinity]) {
    await assert.rejects(createGame(center, size), /Cell size/);
  }
});

test('generated load client requests the saved session ID', async (t) => {
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    assert.equal(new URL(request.url).pathname, `/api/games/${game.sessionId}`);
    assert.equal(request.method, 'GET');
    return Response.json(game);
  });
  assert.deepEqual(await loadGame(game.sessionId), game);
});

test('health check also uses the generated client', async (t) => {
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    assert.equal(new URL(request.url).pathname, '/health');
    return Response.json({ status: 'ok', service: 'catch-me-backend' });
  });
  assert.deepEqual(await checkBackend(), { status: 'ok', service: 'catch-me-backend' });
});

test('API errors become readable errors for the screen', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => Response.json({ error: 'game not found' }, { status: 404 }));
  await assert.rejects(loadGame(game.sessionId), /game not found/);
});

test('malformed successful responses are rejected at runtime', async (t) => {
  for (const body of [null, {}, { ...game, sessionId: 'bad' }, { ...game, createdAt: 'bad' },
    { ...game, playground: { ...game.playground, rows: 8 } },
    { ...game, playground: { ...game.playground, cellSizeMeters: 1550 } },
    { ...game, playground: { ...game.playground, cellSizeMeters: '2000' } },
    { ...game, playground: { ...game.playground, center: { latitude: 81, longitude: 0 } } },
    { ...game, playground: { ...game.playground, bounds: { ...game.playground.bounds, east: 'bad' } } },
    { ...game, playground: { ...game.playground, bounds: { ...game.playground.bounds, north: 0 } } },
  ]) {
    t.mock.method(globalThis, 'fetch', async () => Response.json(body));
    await assert.rejects(createGame(center), /Unexpected game response/);
  }
});

test('load rejects a response for a different session', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => Response.json({ ...game, sessionId: 'f'.repeat(32) }));
  await assert.rejects(loadGame(game.sessionId), /different game/);
});

test('invalid IDs are rejected before a request', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => { assert.fail('should not fetch'); });
  await assert.rejects(loadGame('../health'), /32 lowercase hexadecimal/);
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
  await assert.rejects(createGame(center), /Connection timed out/);
});
