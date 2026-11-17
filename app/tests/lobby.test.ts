import assert from 'node:assert/strict';
import { test } from 'node:test';

import type { GameStateHunterResponse, GameStateRunnerResponse } from '../src/api/types';
import { MARKER_COLORS, ROUTE_COLORS, SKETCH_COLOR, ZONE_COLORS, fakePositionMarkers, formatDuration, hunterTeams, jokerMarkers, jokerZones, playerMarkers, gameHighlights, gameMarkers, routeHighlights, routeChangeBlocker, routeMarkers, playersByTeam, revealMarkers, secondsUntil, secondsUntilStart, sketchLines, thinPoints } from '../src/lobby';
import { cellBounds, previewPlayground } from '../src/playground';

const state: GameStateRunnerResponse = {
  sessionId: '00112233445566778899aabbccddeeff',
  status: 'countdown',
  serverTime: '2026-11-08T12:00:00Z',
  startsAt: '2026-11-08T12:00:10Z',
  playground: previewPlayground({ latitude: 52.52, longitude: 13.405 }),
  playerId: 'a'.repeat(32),
  team: 'runner',
  joinCodes: { runner: '482913', hunter: '071564' },
  players: [
    { playerId: 'a'.repeat(32), name: 'Ada', team: 'runner', ready: true },
    { playerId: 'b'.repeat(32), name: 'Bob', team: 'hunter', ready: true },
    { playerId: 'c'.repeat(32), name: 'Cy', team: 'runner', ready: false },
  ],
  route: {
    start: { title: 'Start', coordinates: { latitude: 52.47, longitude: 13.4 } },
    stop: { title: 'Stop', coordinates: { latitude: 52.57, longitude: 13.4 } },
    distanceMeters: 11119,
  },
  routeRegenerationsLeft: 2,
  jokers: [],
};

const { route: _route, routeRegenerationsLeft: _routeRegenerationsLeft, ...hunterBase } = state;
const hunterState: GameStateHunterResponse = { ...hunterBase, team: 'hunter', jokers: [], sketches: [] };

test('the route can only change in the lobby, with changes left and no runner ready', () => {
  const unready = state.players.map((p) => ({ ...p, ready: false }));
  const lobby: GameStateRunnerResponse = { ...state, status: 'lobby', startsAt: undefined, players: unready };
  assert.equal(routeChangeBlocker(lobby), null);
  assert.equal(routeChangeBlocker({ ...lobby, routeRegenerationsLeft: 0 }), 'noneLeft');
  assert.equal(routeChangeBlocker({ ...lobby, players: state.players }), 'runnerReady');
  // A ready hunter does not block it.
  assert.equal(routeChangeBlocker({ ...lobby, players: unready.map((p) => ({ ...p, ready: p.team === 'hunter' })) }), null);
  assert.equal(routeChangeBlocker(state), 'notLobby');
});

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

test('hunter teams are listed once, in join order', () => {
  const hunter = { team: 'hunter' as const, ready: false };
  assert.deepEqual(hunterTeams([
    ...state.players,
    { ...hunter, playerId: 'd'.repeat(32), name: 'Dan', hunterTeam: 'Wolves' },
    { ...hunter, playerId: 'e'.repeat(32), name: 'Eve', hunterTeam: 'Bears' },
    { ...hunter, playerId: 'f'.repeat(32), name: 'Fay', hunterTeam: 'Wolves' },
  ]), ['Wolves', 'Bears']);
});

test('reveal countdown and duration formatting', () => {
  const running = { ...state, status: 'running' as const, nextRevealAt: '2026-11-08T12:05:10Z' };
  assert.equal(secondsUntil(running.nextRevealAt, running, 0, 0), 310);
  assert.equal(secondsUntil(undefined, running, 0, 0), null);
  assert.equal(formatDuration(310), '5:10');
  assert.equal(formatDuration(9), '0:09');
});

test('markers cover only live positions the server sent, colored by kind', () => {
  const at = '2026-11-08T12:00:00Z';
  const markers = playerMarkers({
    ...state,
    players: [
      { ...state.players[0], position: { latitude: 52.5, longitude: 13.4, accuracyMeters: 7.6, recordedAt: at } },
      { ...state.players[1] },
      { ...state.players[2], position: { latitude: 52.51, longitude: 13.41, accuracyMeters: 5, recordedAt: at } },
    ],
  });
  assert.deepEqual(markers.map((m) => [m.title, m.color]), [
    ['Ada (you)', MARKER_COLORS.me],
    ['Cy', MARKER_COLORS.teammate],
  ]);
  assert.match(markers[0].description, /±8 m/);
});

test('every reveal is a numbered marker per runner, oldest first', () => {
  const at = (n: number) => `2026-11-08T12:0${n}:00Z`;
  const reveal = (n: number) => ({ kind: 'position' as const, position: { latitude: 50 + n, longitude: 6, accuracyMeters: 5, recordedAt: at(n) }, revealedAt: at(n + 1) });
  const markers = revealMarkers({
    ...state,
    players: [
      { ...state.players[0], reveals: [reveal(1), reveal(2), reveal(3)] },
      { ...state.players[2], reveals: [reveal(4), reveal(5)] },
    ],
  });
  assert.deepEqual(markers.map((m) => m.coordinate.latitude), [51, 52, 53, 54, 55]);
  assert.deepEqual(markers.map((m) => m.label), ['1', '2', '3', '1', '2']);
  assert.ok(markers.every((m) => m.color === MARKER_COLORS.revealed));
  assert.equal(new Set(markers.map((m) => m.id)).size, 5);
  assert.match(markers[2].description, /Revealed at/);
  assert.deepEqual(revealMarkers(state), []);
});

test('a zone reveal is highlighted and marked in the middle of its cell', () => {
  const at = (n: number) => `2026-11-08T12:0${n}:00Z`;
  const exact = { kind: 'position' as const, position: { latitude: 52.5, longitude: 13.4, accuracyMeters: 5, recordedAt: at(1) }, revealedAt: at(2) };
  const zone = { kind: 'zone' as const, cell: 'C4', revealedAt: at(3) };
  const outside = { kind: 'outside' as const, revealedAt: at(4) };
  const running = {
    ...hunterState,
    players: [{ ...state.players[0], reveals: [exact, zone, outside, { ...exact, revealedAt: at(5) }] }, state.players[1]],
  };
  const markers = revealMarkers(running);
  // The reveal outside the playground has no marker but keeps its number.
  assert.deepEqual(markers.map((m) => m.label), ['1', '2', '4']);
  const b = cellBounds(state.playground, 'C4');
  assert.deepEqual(markers[1].coordinate, { latitude: (b.north + b.south) / 2, longitude: (b.east + b.west) / 2 });
  assert.match(markers[1].description, /Only cell C4/);
  assert.deepEqual(gameHighlights(running).map((h) => [h.cell, h.title, h.color]), [['C4', 'Ada', ZONE_COLORS.revealed]]);
});

test('a used joker becomes one marker per runner it showed', () => {
  const position = (playerId: string, latitude: number) => ({ playerId, latitude, longitude: 13.4, accuracyMeters: 5, recordedAt: '2026-11-08T12:06:58Z' });
  const markers = jokerMarkers({
    ...hunterState,
    jokers: [{ kind: 'showRunners', available: false, usedAt: '2026-11-08T12:07:00Z', positions: [position(state.players[0].playerId, 52.5), position(state.players[2].playerId, 52.6)] }],
  });
  assert.deepEqual(markers.map((m) => [m.title, m.coordinate.latitude, m.color]), [
    ['Ada', 52.5, MARKER_COLORS.joker],
    ['Cy', 52.6, MARKER_COLORS.joker],
  ]);
  assert.match(markers[0].description, /Joker/);
  assert.deepEqual(jokerMarkers({ ...hunterState, jokers: [{ kind: 'showRunners', available: true }] }), []);
});

test('a queried zone is highlighted and marked in its center, colored by the answer', () => {
  const jokers = [
    { kind: 'queryZone' as const, available: false, usedAt: '2026-11-08T12:07:00Z', cell: 'A1', found: true },
    { kind: 'queryZone' as const, available: false, usedAt: '2026-11-08T12:08:00Z', cell: 'F6', found: false },
  ];
  const zones = jokerZones({ ...hunterState, jokers });
  assert.deepEqual(zones.map((z) => [z.cell, z.color]), [['A1', ZONE_COLORS.found], ['F6', ZONE_COLORS.empty]]);
  assert.match(zones[0].title, /at least one runner/);
  assert.match(zones[1].title, /no runner/);
  const [marker] = jokerMarkers({ ...hunterState, jokers: [jokers[0]] });
  const b = cellBounds(state.playground, 'A1');
  assert.deepEqual(marker.coordinate, { latitude: (b.north + b.south) / 2, longitude: (b.east + b.west) / 2 });
  assert.equal(marker.color, ZONE_COLORS.found);
  assert.deepEqual(jokerZones({ ...hunterState, jokers: [{ kind: 'queryZone', available: true }] }), []);
});

test('the fake position is a marker for every runner, once used', () => {
  const point = { latitude: 52.53, longitude: 13.41 };
  const [marker, ...rest] = fakePositionMarkers({
    ...state,
    jokers: [{ kind: 'fakePosition', available: false, usedAt: '2026-11-08T12:07:00Z', usedBy: state.players[0].playerId, point }],
  });
  assert.equal(rest.length, 0);
  assert.deepEqual(marker.coordinate, point);
  assert.equal(marker.color, MARKER_COLORS.joker);
  assert.match(marker.title, /Ada/);
  assert.deepEqual(fakePositionMarkers({ ...state, jokers: [{ kind: 'fakePosition', available: true }] }), []);
});

test('sketches become lines in the sketch color, in the order the server sent them', () => {
  const points = [{ latitude: 52.5, longitude: 13.4 }, { latitude: 52.51, longitude: 13.41 }];
  const sketches = [
    { sketchId: '1'.repeat(32), createdAt: '2026-11-08T12:07:00Z', points },
    { sketchId: '2'.repeat(32), createdAt: '2026-11-08T12:08:00Z', points: [...points].reverse() },
  ];
  assert.deepEqual(sketchLines({ ...hunterState, sketches }), [
    { id: '1'.repeat(32), coordinates: points, color: SKETCH_COLOR },
    { id: '2'.repeat(32), coordinates: [...points].reverse(), color: SKETCH_COLOR },
  ]);
  assert.deepEqual(sketchLines(hunterState), []);
});

test('long lines are thinned evenly, keeping their ends', () => {
  const points = Array.from({ length: 1201 }, (_, i) => i);
  const thinned = thinPoints(points, 500);
  assert.equal(thinned.length, 500);
  assert.equal(thinned[0], 0);
  assert.equal(thinned[499], 1200);
  assert.ok(thinned.every((p, i) => i === 0 || p > thinned[i - 1]), 'order kept, no duplicates');
  assert.equal(thinPoints([1, 2, 3], 500).length, 3);
  assert.deepEqual(thinPoints([1, 2, 3, 4, 5], 2), [1, 5]);
});

test('the route shows as start and target cells and markers, with generic names for untitled stops', () => {
  const route = { ...state.route, stop: { ...state.route.stop, title: '' } };
  const highlights = routeHighlights(state.playground, route);
  assert.deepEqual(highlights.map((h) => h.color), [ROUTE_COLORS.start, ROUTE_COLORS.target]);
  assert.equal(highlights[0].title, 'Start');
  assert.notEqual(highlights[1].title, '');
  assert.deepEqual(routeMarkers(route).map((m) => [m.id, m.color]), [['route-start', ROUTE_COLORS.start], ['route-stop', ROUTE_COLORS.target]]);
  const outside = { ...state.route, start: { ...state.route.start, coordinates: { latitude: 0, longitude: 0 } } };
  assert.deepEqual(routeHighlights(state.playground, outside).map((h) => h.color), [ROUTE_COLORS.target]);
});

test('runners see the route, hunters the queried zones, and nobody sees their own position as a marker', () => {
  assert.equal(gameHighlights(state).length, 2);
  assert.deepEqual(gameHighlights(hunterState), []);
  assert.deepEqual(gameMarkers(state).map((m) => m.id).slice(0, 2), ['route-start', 'route-stop']);
  assert.ok(!gameMarkers(hunterState).some((m) => m.id === hunterState.playerId));
});
