import assert from 'node:assert/strict';
import { test } from 'node:test';

import type { GameResultResponse, ResultPlayerResponse } from '../src/api/types';
import { SPECTATOR_COLORS } from '../src/lobby';
import { positionAt, replayLegend, replayTracks, tailAt } from '../src/replay';

const points = [
  { latitude: 52, longitude: 13, at: 10_000 },
  { latitude: 53, longitude: 14, at: 20_000 },
  { latitude: 53, longitude: 16, at: 60_000 },
];

test('a player is hidden before the first point, interpolated between points and stays at the last one', () => {
  assert.equal(positionAt(points, 9_999), null);
  assert.deepEqual(positionAt(points, 10_000), { latitude: 52, longitude: 13 });
  assert.deepEqual(positionAt(points, 15_000), { latitude: 52.5, longitude: 13.5 });
  assert.deepEqual(positionAt(points, 40_000), { latitude: 53, longitude: 15 });
  assert.deepEqual(positionAt(points, 90_000), { latitude: 53, longitude: 16 });
  assert.equal(positionAt([], 0), null);
});

test('the tail covers only the given time before now and ends at the current position', () => {
  assert.deepEqual(tailAt(points, 5_000), []);
  // Whole track so far, since the tail reaches back before the first point.
  assert.deepEqual(tailAt(points, 15_000, 60_000), [{ latitude: 52, longitude: 13 }, { latitude: 52.5, longitude: 13.5 }]);
  // Starts mid-segment 25 s before now.
  assert.deepEqual(tailAt(points, 40_000, 25_000), [{ latitude: 52.5, longitude: 13.5 }, { latitude: 53, longitude: 14 }, { latitude: 53, longitude: 15 }]);
});

test('tracks of runners and hunters are timed from the start; spectators have none', () => {
  const result = {
    startsAt: '2026-11-18T12:00:00Z',
    players: [
      { playerId: 'a'.repeat(32), name: 'Ada', team: 'runner', runner: { status: 'caught', endedAt: '2026-11-18T12:30:00Z' }, track: [{ latitude: 52, longitude: 13, accuracyMeters: 5, recordedAt: '2026-11-18T12:00:05Z' }] },
      { playerId: 'b'.repeat(32), name: 'Bob', team: 'hunter', hunterTeam: 'Pack', track: [] },
      { playerId: 'c'.repeat(32), name: 'Viewer', team: 'spectator' },
    ],
  } as GameResultResponse;
  const [pack] = replayLegend(result).filter((entry) => entry.team === 'hunter');
  assert.deepEqual(replayTracks(result), [
    { playerId: 'a'.repeat(32), name: 'Ada', color: SPECTATOR_COLORS.runner, points: [{ latitude: 52, longitude: 13, at: 5_000 }] },
    { playerId: 'b'.repeat(32), name: 'Bob', color: pack.color, points: [] },
  ]);
});

test('each hunter team gets its own color in join order, runners stay blue', () => {
  const hunter = (id: string, hunterTeam?: string): ResultPlayerResponse => ({ playerId: id.repeat(32), name: id, team: 'hunter', ...(hunterTeam ? { hunterTeam } : {}), track: [] });
  const players: ResultPlayerResponse[] = [hunter('b', 'Wolves'), { playerId: 'a'.repeat(32), name: 'Ada', team: 'runner', track: [] }, hunter('c', 'Pack'), hunter('d', 'Wolves'), hunter('e')];
  const result = { startsAt: '2026-11-18T12:00:00Z', players } as GameResultResponse;
  const legend = replayLegend(result);
  assert.deepEqual(legend.map(({ key, team, hunterTeam }) => ({ key, team, hunterTeam })), [
    { key: 'runner', team: 'runner', hunterTeam: undefined },
    { key: 'hunter:Wolves', team: 'hunter', hunterTeam: 'Wolves' },
    { key: 'hunter:Pack', team: 'hunter', hunterTeam: 'Pack' },
    { key: 'hunter:', team: 'hunter', hunterTeam: undefined },
  ]);
  assert.equal(legend[0].color, SPECTATOR_COLORS.runner);
  assert.equal(new Set(legend.map((entry) => entry.color)).size, legend.length);
  const colors = replayTracks(result).map((track) => track.color);
  assert.equal(colors[0], colors[3], 'members of one team share its color');
  assert.notEqual(colors[0], colors[2]);
});
