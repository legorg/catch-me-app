import assert from 'node:assert/strict';
import { test } from 'node:test';

import { API_URL, checkBackend, loadDemoGreeting } from '../src/api/client';

test('generated greeting client sends the name as an encoded query parameter', async (t) => {
  const name = 'Runner & Hunter';
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    const url = new URL(request.url);
    assert.equal(url.origin, new URL(API_URL).origin);
    assert.equal(url.pathname, '/api/demo');
    assert.equal(url.searchParams.get('name'), name);
    assert.equal(request.method, 'GET');
    return Response.json({
      message: `Hello, ${name}! Your Expo app is connected to Go.`,
      playerName: name,
      service: 'catch-me-backend',
    });
  });

  const greeting = await loadDemoGreeting(name);
  assert.equal(greeting.playerName, name);
  assert.equal(greeting.message, `Hello, ${name}! Your Expo app is connected to Go.`);
});

test('health check also uses the generated client', async (t) => {
  t.mock.method(globalThis, 'fetch', async (request: Request) => {
    assert.equal(new URL(request.url).pathname, '/health');
    return Response.json({ status: 'ok', service: 'catch-me-backend' });
  });
  assert.deepEqual(await checkBackend(), { status: 'ok', service: 'catch-me-backend' });
});

test('API errors become readable errors for the screen', async (t) => {
  t.mock.method(globalThis, 'fetch', async () =>
    Response.json({ error: 'name must be valid UTF-8 and at most 60 characters' }, { status: 400 }),
  );
  await assert.rejects(loadDemoGreeting('x'.repeat(61)), /at most 60 characters/);
});

test('malformed successful responses are rejected at runtime', async (t) => {
  t.mock.method(globalThis, 'fetch', async () =>
    Response.json({ message: 123, playerName: 'Runner', service: 'catch-me-backend' }),
  );
  await assert.rejects(loadDemoGreeting(), /Unexpected demo response/);
});

test('network failures are propagated', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => {
    throw new TypeError('Network request failed');
  });
  await assert.rejects(loadDemoGreeting(), /Network request failed/);
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
  await assert.rejects(loadDemoGreeting(), /Connection timed out/);
});
