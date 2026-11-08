import assert from 'node:assert/strict';
import { test } from 'node:test';

import type { GameStateResponse } from '../src/api/client';
import { playersByTeam, secondsUntilStart } from '../src/lobby';
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
