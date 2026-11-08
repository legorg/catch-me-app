import assert from 'node:assert/strict';
import { test } from 'node:test';

import type { GameStateResponse } from '../src/api/client';
import { MARKER_COLORS, formatDuration, playerMarkers, playersByTeam, secondsUntil, secondsUntilStart } from '../src/lobby';
import { previewPlayground } from '../src/playground';

const state: GameStateResponse = {
  sessionId: '00112233445566778899aabbccddeeff',
  status: 'countdown',
  serverTime: '2026-11-08T12:00:00Z',
  startsAt: '2026-11-08T12:00:10Z',
  playground: previewPlayground({ latitude: 52.52, longitude: 13.405 }),
  playerId: 'a'.repeat(32),
  team: 'runner',
  joinCode: 'K7M2QX',
  players: [
    { playerId: 'a'.repeat(32), name: 'Ada', team: 'runner', ready: true },
    { playerId: 'b'.repeat(32), name: 'Bob', team: 'hunter', ready: true },
    { playerId: 'c'.repeat(32), name: 'Cy', team: 'runner', ready: false },
  ],
};

test('countdown uses server time plus local elapsed time', () => {
  // The phone clock is far off; only the difference since receipt matters.
  const receivedAt = 1_000_000;
  assert.equal(secondsUntilStart(state, receivedAt, receivedAt), 10);
  assert.equal(secondsUntilStart(state, receivedAt, receivedAt + 2500), 8);
  assert.equal(secondsUntilStart(state, receivedAt, receivedAt + 9001), 1);
  assert.equal(secondsUntilStart(state, receivedAt, receivedAt + 60_000), 0);
  assert.equal(secondsUntilStart({ ...state, startsAt: undefined, status: 'lobby' }, receivedAt, receivedAt), null);
});

test('players are grouped by team in join order', () => {
  const teams = playersByTeam(state.players);
  assert.deepEqual(teams.runner.map((p) => p.name), ['Ada', 'Cy']);
  assert.deepEqual(teams.hunter.map((p) => p.name), ['Bob']);
});

test('reveal countdown and duration formatting', () => {
  const running = { ...state, status: 'running' as const, nextRevealAt: '2026-11-08T12:05:10Z' };
  assert.equal(secondsUntil(running.nextRevealAt, running, 0, 0), 310);
  assert.equal(secondsUntil(undefined, running, 0, 0), null);
  assert.equal(formatDuration(310), '5:10');
  assert.equal(formatDuration(9), '0:09');
});

test('markers cover only positions the server sent, colored by kind', () => {
  const at = '2026-11-08T12:00:00Z';
  const markers = playerMarkers({
    ...state,
    players: [
      { ...state.players[0], position: { latitude: 52.5, longitude: 13.4, accuracyMeters: 7.6, recordedAt: at } },
      { ...state.players[1] },
      { ...state.players[2], position: { latitude: 52.51, longitude: 13.41, accuracyMeters: 5, recordedAt: at, revealedAt: at } },
    ],
  });
  assert.deepEqual(markers.map((m) => [m.title, m.color]), [
    ['Ada (you)', MARKER_COLORS.me],
    ['Cy', MARKER_COLORS.revealed],
  ]);
  assert.match(markers[0].description, /±8 m/);
  assert.match(markers[1].description, /Revealed at/);
});