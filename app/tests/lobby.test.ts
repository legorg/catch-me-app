import assert from 'node:assert/strict';
import { test } from 'node:test';

import type { GameStateHunterResponse, GameStateRunnerResponse } from '../src/api/types';
import { MARKER_COLORS, ROUTE_COLORS, SKETCH_COLOR, ZONE_COLORS, caughtMarkers, compassLines, compassMarkers, fakePositionMarkers, formatDuration, hunterTeams, jokerMarkers, jokerZones, playerMarkers, gameHighlights, gameMarkers, routeHighlights, routeChangeBlocker, routeMarkers, playersByTeam, revealMarkers, secondsUntil, secondsUntilStart, shownHunterMarkers, sketchLines, thinPoints } from '../src/lobby';
import { canFinish, canShareGameLocation, gameDestination, isActivePlayer, shouldAcceptGameState } from '../src/game-state';
import { gameResultRoute } from '../src/game-result';
import { cellBounds, distanceMeters, offsetPoint, previewPlayground } from '../src/playground';

const state: GameStateRunnerResponse = {
  sessionId: '00112233445566778899aabbccddeeff',
  status: 'countdown',
  serverTime: '2026-11-08T12:00:00Z',
  startsAt: '2026-11-08T12:00:10Z',
  revealIntervalMinutes: 30,
  playground: previewPlayground({ latitude: 52.52, longitude: 13.405 }),
  playerId: 'a'.repeat(32),
  team: 'runner',
  joinCodes: { runner: '482913', hunter: '071564' },
  players: [
    { playerId: 'a'.repeat(32), name: 'Ada', team: 'runner', ready: true, runner: { status: 'active' } },
    { playerId: 'b'.repeat(32), name: 'Bob', team: 'hunter', ready: true },
    { playerId: 'c'.repeat(32), name: 'Cy', team: 'runner', ready: false, runner: { status: 'active' } },
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

test('ended games always lead to the same ID-addressed result, never back to the lobby or game', () => {
  const expected = { pathname: '/results/[sessionId]', params: { sessionId: state.sessionId } };
  assert.deepEqual(gameResultRoute(state.sessionId), expected);
  assert.equal(gameDestination({ ...state, status: 'lobby' }, 0), '/lobby');
  assert.equal(gameDestination(state, 10), '/lobby');
  assert.equal(gameDestination(state, 0), '/game');
  assert.equal(gameDestination({ ...state, status: 'running' }), '/game');
  for (const current of [state, hunterState]) {
    const ended = { ...current, status: 'ended' as const, endedAt: '2026-11-18T13:00:00Z' };
    assert.deepEqual(gameDestination(ended), expected); // Restored session.
    assert.deepEqual(gameDestination(ended, 0), expected); // Elapsed countdown must not restart it.
    assert.equal(canShareGameLocation(ended), false); // Hunters stop too.
    assert.equal(canFinish(ended, 0, 0), false);
    assert.equal(shouldAcceptGameState(ended, { ...current, status: 'running', serverTime: '2026-11-18T14:00:00Z' }), false);
    assert.equal(shouldAcceptGameState(ended, { ...ended, serverTime: '2026-11-18T14:00:00Z' }), true);
  }
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

test('a catch is one black marker for everyone, including the caught runner, without waiting for a ping', () => {
  const endedAt = '2026-11-08T12:01:00Z';
  const position = { latitude: 52.51, longitude: 13.41, accuracyMeters: 5, recordedAt: '2026-11-08T12:00:58Z' };
  const caught = { ...state.players[0], runner: { status: 'caught' as const, endedAt, caughtPosition: position } };
  for (const viewer of [state, { ...state, playerId: state.players[2].playerId }, { ...hunterState, playerId: state.players[1].playerId }]) {
    const snapshot = { ...viewer, players: [caught, state.players[1], state.players[2]] };
    const markers = caughtMarkers(snapshot);
    assert.equal(markers.length, 1);
    assert.equal(markers[0].color, '#000000');
    assert.equal(markers[0].label, '×');
    assert.deepEqual(markers[0].coordinate, { latitude: position.latitude, longitude: position.longitude });
    assert.match(markers[0].title, /Ada.*Caught/);
    assert.match(markers[0].description, /last reported position/);
    assert.ok(gameMarkers(snapshot).some((marker) => marker.id === markers[0].id));
  }
  // No report (or expired history) means status only, not a made-up marker.
  assert.deepEqual(caughtMarkers({ ...state, players: [{ ...caught, runner: { ...caught.runner, caughtPosition: undefined } }] }), []);
});

test('catch markers replace live, ping, zone-reveal and historical joker markers for that runner', () => {
  const at = '2026-11-08T12:01:00Z';
  const position = { latitude: 52.51, longitude: 13.41, accuracyMeters: 5, recordedAt: at };
  const caught = {
    ...state.players[0], runner: { status: 'caught' as const, endedAt: at, caughtPosition: position },
    // Even stale marker data must not draw this runner as still playing.
    position,
    reveals: [{ kind: 'position' as const, revealedAt: at, position }, { kind: 'zone' as const, revealedAt: at, cell: 'C4' }],
  };
  const players = [caught, state.players[1], { ...state.players[2], position }];
  const hunters: GameStateHunterResponse = {
    ...hunterState, players,
    jokers: [{ kind: 'showRunners', available: false, usedAt: at, positions: [
      { ...position, playerId: caught.playerId }, { ...position, playerId: state.players[2].playerId },
    ] }],
  };
  assert.deepEqual(revealMarkers(hunters), []);
  assert.deepEqual(gameHighlights(hunters), []);
  assert.deepEqual(playerMarkers(hunters).map((marker) => marker.id), [state.players[2].playerId]);
  assert.deepEqual(jokerMarkers(hunters).map((marker) => marker.title), ['Cy']);
  assert.deepEqual(gameMarkers(hunters).map((marker) => marker.id), [
    `joker-showRunners-${state.players[2].playerId}`, state.players[2].playerId, `caught-${caught.playerId}`,
  ]);
  const runners: GameStateRunnerResponse = {
    ...state, players, jokers: [{ kind: 'fakePosition', available: false, usedAt: at, usedBy: caught.playerId, point: position }],
  };
  assert.deepEqual(fakePositionMarkers(runners), []);
});

test('only active players with a running game may share game locations', () => {
  const running: GameStateRunnerResponse = { ...state, status: 'running' };
  const caught = { ...running, players: [{ ...running.players[0], runner: { status: 'caught' as const, endedAt: running.serverTime } }] };
  const finished = { ...running, players: [{ ...running.players[0], runner: { status: 'finished' as const, endedAt: running.serverTime } }] };
  assert.equal(canShareGameLocation(undefined), false);
  assert.equal(canShareGameLocation(state), false);
  assert.equal(canShareGameLocation({ ...state, status: 'lobby' }), false);
  assert.equal(canShareGameLocation(running), true);
  assert.equal(canShareGameLocation({ ...hunterState, playerId: state.players[1].playerId, status: 'running' }), true);
  assert.equal(canShareGameLocation(caught), false);
  assert.equal(canShareGameLocation(finished), false);
  assert.equal(canShareGameLocation({ ...running, players: [] }), false);
  assert.equal(isActivePlayer(undefined), false);
  assert.equal(isActivePlayer({ ...running.players[0], runner: undefined }), false);
});

test('finished runners have no ordinary, historical joker, fake-position or catch markers', () => {
  const at = state.serverTime;
  const position = { latitude: 52.51, longitude: 13.41, accuracyMeters: 5, recordedAt: at };
  const finished = {
    ...state.players[0], runner: { status: 'finished' as const, endedAt: at },
    // Even stale data is suppressed by status, not by timestamp presence.
    position, reveals: [{ kind: 'position' as const, revealedAt: at, position }],
  };
  const players = [finished, state.players[1]];
  const runners: GameStateRunnerResponse = { ...state, players, jokers: [{ kind: 'fakePosition', available: false, usedBy: finished.playerId, point: position }] };
  const hunters: GameStateHunterResponse = { ...hunterState, players, jokers: [{ kind: 'showRunners', available: false, positions: [{ ...position, playerId: finished.playerId }] }] };
  assert.deepEqual(playerMarkers(runners), []);
  assert.deepEqual(revealMarkers(hunters), []);
  assert.deepEqual(caughtMarkers(runners), []);
  assert.deepEqual(jokerMarkers(hunters), []);
  assert.deepEqual(fakePositionMarkers(runners), []);
});

test('finishing requires a recent server-confirmed report within 50 m of the stop', () => {
  const receivedAt = Date.UTC(2040, 0, 1); // The phone's absolute clock is irrelevant.
  const report = { ...state.route.stop.coordinates, accuracyMeters: 5, recordedAt: state.serverTime };
  const running: GameStateRunnerResponse = { ...state, status: 'running', players: [{ ...state.players[0], position: report }] };
  assert.equal(canFinish(running, receivedAt, receivedAt), true);
  assert.equal(canFinish(running, receivedAt, receivedAt + 30_000), true);
  assert.equal(canFinish(running, receivedAt, receivedAt + 30_001), false);
  assert.equal(canFinish(running, receivedAt, receivedAt - 250), true); // Render tick just predates receipt.
  for (const meters of [49.99, 50.01]) {
    const position = { ...report, ...offsetPoint(state.route.stop.coordinates, 0, meters) };
    assert.equal(canFinish({ ...running, players: [{ ...running.players[0], position }] }, receivedAt, receivedAt), meters < 50);
  }
  for (const position of [
    undefined,
    { ...report, ...state.route.start.coordinates },
    { ...report, recordedAt: new Date(Date.parse(state.serverTime) - 30_001).toISOString() },
    { ...report, recordedAt: 'not a date' },
    { ...report, recordedAt: new Date(Date.parse(state.serverTime) + 1000).toISOString() },
  ]) {
    assert.equal(canFinish({ ...running, players: [{ ...running.players[0], position }] }, receivedAt, receivedAt), false);
  }
  assert.equal(running.players[0].runner?.status, 'active'); // Eligibility never finishes automatically.
});

test('the existing test override skips only the finish position gate', () => {
  const running: GameStateRunnerResponse = { ...state, status: 'running', startPositionCheckDisabled: true };
  assert.equal(canFinish(running, 0, 0), true); // No position or permission needed for stationary testing.
  const stale = { ...state.route.start.coordinates, accuracyMeters: 5, recordedAt: '2000-01-01T00:00:00Z' };
  assert.equal(canFinish({ ...running, players: [{ ...state.players[0], position: stale }] }, 0, 0), true);
  for (const blocked of [
    undefined, state, { ...running, status: 'lobby' as const }, { ...running, status: 'countdown' as const },
    { ...hunterState, status: 'running' as const, startPositionCheckDisabled: true },
    { ...running, players: [] },
    ...(['caught', 'finished'] as const).map((status) => ({ ...running, players: [{ ...state.players[0], runner: { status, endedAt: state.serverTime } }] })),
  ]) {
    assert.equal(canFinish(blocked, 0, 0), false);
  }
});

test('late responses cannot undo or change a confirmed runner outcome', () => {
  const active: GameStateRunnerResponse = { ...state, status: 'running' };
  const caught = { ...active, players: [{ ...active.players[0], runner: { status: 'caught' as const, endedAt: active.serverTime } }] };
  const finished = { ...active, players: [{ ...active.players[0], runner: { status: 'finished' as const, endedAt: active.serverTime } }] };
  assert.equal(shouldAcceptGameState(undefined, active), true);
  assert.equal(shouldAcceptGameState(active, caught), true);
  assert.equal(shouldAcceptGameState(caught, active), false);
  assert.equal(shouldAcceptGameState(caught, { ...active, serverTime: '2026-11-08T12:02:00Z' }), false);
  assert.equal(shouldAcceptGameState(caught, { ...caught, serverTime: '2026-11-08T12:02:00Z' }), true);
  assert.equal(shouldAcceptGameState(active, { ...active, serverTime: '2026-11-08T11:59:59Z' }), false);
  assert.equal(shouldAcceptGameState(caught, { ...active, sessionId: 'd'.repeat(32) }), true);
  assert.equal(shouldAcceptGameState(active, finished), true);
  assert.equal(shouldAcceptGameState(finished, active), false);
  assert.equal(shouldAcceptGameState(finished, caught), false);
  assert.equal(shouldAcceptGameState(caught, finished), false);
  assert.equal(shouldAcceptGameState(finished, { ...finished, serverTime: '2026-11-08T12:02:00Z' }), true);
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

test('the personal fake position is a marker for its runner, once used', () => {
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

test('the shown hunters are markers in their own color for every runner, once used', () => {
  const position = { playerId: state.players[1].playerId, latitude: 52.51, longitude: 13.4, accuracyMeters: 5, recordedAt: '2026-11-08T12:06:58Z' };
  const running = {
    ...state,
    jokers: [{ kind: 'showHunters' as const, available: false, usedAt: '2026-11-08T12:07:00Z', usedBy: state.players[0].playerId, positions: [position] }],
  };
  assert.deepEqual(shownHunterMarkers(running).map((m) => [m.title, m.coordinate.latitude, m.color]), [['Bob', 52.51, MARKER_COLORS.shownHunter]]);
  assert.notEqual(MARKER_COLORS.shownHunter, MARKER_COLORS.joker);
  assert.ok(gameMarkers(running).some((m) => m.id === `joker-showHunters-${position.playerId}`));
  assert.deepEqual(shownHunterMarkers({ ...state, jokers: [{ kind: 'showHunters', available: true }] }), []);
});

test('the compass becomes one cell-long arrow per direction from the origin, without names', () => {
  const origin = { latitude: 52.52, longitude: 13.405 };
  const running = {
    ...hunterState,
    jokers: [{ kind: 'compass' as const, available: false, usedAt: '2026-11-08T12:07:00Z', origin, bearings: [0, 90] }],
  };
  const [north, east, ...rest] = compassLines(running);
  assert.equal(rest.length, 0);
  assert.deepEqual(north.coordinates[0], origin);
  assert.equal(north.color, MARKER_COLORS.joker);
  // The tip lies one cell away in the given direction.
  const tipNorth = north.coordinates[1];
  assert.ok(Math.abs(distanceMeters(origin, tipNorth) - state.playground.cellSizeMeters) < 1);
  assert.ok(tipNorth.latitude > origin.latitude && Math.abs(tipNorth.longitude - origin.longitude) < 1e-9);
  const tipEast = east.coordinates[1];
  assert.ok(tipEast.longitude > origin.longitude && Math.abs(tipEast.latitude - origin.latitude) < 1e-9);
  // Shaft, then both barbs from the tip.
  assert.equal(north.coordinates.length, 5);
  assert.deepEqual(north.coordinates[3], tipNorth);
  const [marker] = compassMarkers(running);
  assert.deepEqual(marker.coordinate, origin);
  assert.ok(gameMarkers(running).some((m) => m.id === 'joker-compass'));
  assert.deepEqual(compassLines({ ...hunterState, jokers: [{ kind: 'compass', available: true }] }), []);
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
