import assert from 'node:assert/strict';
import { test } from 'node:test';

import type { GameStateHunterResponse, GameStateRunnerResponse, GameStateSpectatorResponse } from '../src/api/types';
import { MARKER_COLORS, ROUTE_COLORS, SKETCH_COLOR, SPECTATOR_COLORS, ZONE_COLORS, caughtMarkers, compassLines, compassMarkers, fakePositionMarkers, formatDuration, hunterTeams, joinInvitationMessage, jokerMarkers, jokerZones, playerMarkers, gameHighlights, gameMarkers, remainingUses, routeHighlights, routeChangeBlocker, routeMarkers, playersByTeam, revealMarkers, secondsUntil, secondsUntilStart, serverNow, shownHunterMarkers, sketchLines, spectatorMarkers, thinPoints, TRAP_COLORS, TRAP_RADIUS_METERS, trapCircles, visibleRoute, ZONE_ALERT_HINT_SECONDS, recentZoneAlert, zoneAlertEntries } from '../src/lobby';
import { canFinish, canShareGameLocation, canShareLobbyLocation, gameDestination, isActivePlayer, isSpectator, shouldAcceptGameState } from '../src/game-state';
import { getLanguage, setLanguage } from '../src/i18n';
import { gameResultRoute } from '../src/game-result';
import { cellBounds, distanceMeters, offsetPoint, previewPlayground } from '../src/playground';

const state: GameStateRunnerResponse = {
  role: 'player',
  sessionId: '00112233445566778899aabbccddeeff',
  status: 'countdown',
  serverTime: '2026-11-08T12:00:00Z',
  startsAt: '2026-11-08T12:00:10Z',
  revealIntervalMinutes: 30,
  jokerSettings: {
    showRunners: { allowance: 1 }, queryZone: { allowance: 1 }, compass: { allowance: 1 },
    trap: { allowance: 1 }, zoneOnly: { allowance: 1 }, fakePosition: { allowance: 1 },
    showHunters: { allowance: 1 }, zoneAlert: { enabled: true },
  },
  playground: previewPlayground({ latitude: 52.52, longitude: 13.405 }),
  playerId: 'a'.repeat(32),
  team: 'runner',
  joinCodes: { runner: '482913', hunter: '071564', spectator: '305821' },
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
  zoneAlerts: [],
};

const { route: _route, routeRegenerationsLeft: _routeRegenerationsLeft, zoneAlerts: _zoneAlerts, ...hunterBase } = state;
const hunterState: GameStateHunterResponse = { ...hunterBase, team: 'hunter', jokers: [], sketches: [] };

const spectatorState: GameStateSpectatorResponse = {
  ...hunterBase, role: 'spectator', team: 'spectator', playerId: 'd'.repeat(32),
  route: state.route,
  players: [...state.players, { playerId: 'd'.repeat(32), name: 'Viewer', team: 'spectator', ready: false }],
  jokers: [],
};

test('only playing lobby participants may track, request location access or ready up', () => {
  assert.equal(canShareLobbyLocation(undefined), false);
  for (const phase of ['lobby', 'countdown'] as const) {
    assert.equal(canShareLobbyLocation({ ...state, status: phase }), true);
    assert.equal(canShareLobbyLocation({ ...hunterState, playerId: state.players[1].playerId, status: phase }), true);
    assert.equal(canShareLobbyLocation({ ...spectatorState, status: phase }), false);
    assert.equal(canShareLobbyLocation({ ...state, role: 'spectator', status: phase }), false); // Even with a stale active player entry.
    assert.equal(canShareLobbyLocation({ ...state, players: [], status: phase }), false);
  }
  for (const status of ['running', 'ended'] as const) {
    assert.equal(canShareLobbyLocation({ ...state, status }), false);
    assert.equal(canShareLobbyLocation({ ...spectatorState, status }), false);
  }
});

test('direct spectators follow lobby, start and result navigation without a playing role', () => {
  assert.equal(gameDestination({ ...spectatorState, status: 'lobby' }), '/lobby');
  assert.equal(gameDestination(spectatorState, 5), '/lobby');
  assert.equal(gameDestination(spectatorState, 0), '/game');
  assert.equal(gameDestination({ ...spectatorState, status: 'running' }), '/game');
  assert.deepEqual(gameDestination({ ...spectatorState, status: 'ended' }, 0), gameResultRoute(state.sessionId));
  for (const status of ['lobby', 'countdown', 'running', 'ended'] as const) {
    const snapshot = { ...spectatorState, status };
    assert.equal(canShareGameLocation(snapshot), false);
    assert.equal(canFinish(snapshot, 0, 0), false);
    assert.deepEqual(gameMarkers(snapshot), routeMarkers(state.route)); // Landmarks, never an invented viewer position.
  }
});

test('direct and converted spectators see the same start/target pins and cells as runners, never hunter route data', () => {
  for (const status of ['lobby', 'countdown', 'running'] as const) {
    for (const viewer of [spectatorState, { ...state, role: 'spectator' as const }]) {
      const snapshot = { ...viewer, status };
      assert.deepEqual(visibleRoute(snapshot), state.route);
      assert.deepEqual(gameMarkers(snapshot), routeMarkers(state.route));
      assert.deepEqual(gameHighlights(snapshot), routeHighlights(state.playground, state.route));
    }
    assert.equal(visibleRoute({ ...hunterState, status }), undefined);
    assert.ok(!gameMarkers({ ...hunterState, status }).some((marker) => marker.id.startsWith('route-')));
  }
  const changedRoute = { ...state.route, start: { ...state.route.start, title: 'New start' } };
  assert.deepEqual(gameMarkers({ ...spectatorState, route: changedRoute }), routeMarkers(changedRoute));
});

test('spectator rosters keep original runner outcomes separate from direct spectators', () => {
  const players = spectatorState.players.map((player) => player.playerId === state.playerId
    ? { ...player, runner: { status: 'finished' as const, endedAt: state.serverTime } } : player);
  const teams = playersByTeam(players);
  assert.deepEqual(teams.spectator.map((player) => player.name), ['Viewer']);
  assert.deepEqual(teams.runner.map((player) => player.name), ['Ada', 'Cy']);
  assert.deepEqual(teams.hunter.map((player) => player.name), ['Bob']);
  assert.equal(teams.runner.length + teams.hunter.length + teams.spectator.length, players.length);
});

test('each localized invitation shares only the selected six-digit code and its affiliation', () => {
  const original = getLanguage();
  try {
    for (const language of ['en', 'de'] as const) {
      setLanguage(language);
      for (const team of ['runner', 'hunter', 'spectator'] as const) {
        const message = joinInvitationMessage(state.joinCodes, team);
        assert.ok(message.includes(state.joinCodes[team]));
        assert.ok(message.includes('Catch me!'));
        assert.ok(!message.includes(state.sessionId));
        assert.equal(message.match(/\d{6}/g)?.length, 1);
        for (const other of ['runner', 'hunter', 'spectator'] as const) {
          if (other !== team) assert.ok(!message.includes(state.joinCodes[other]));
        }
        if (team === 'spectator') assert.match(message, language === 'de' ? /Zuschauer/ : /Spectators/);
      }
    }
  } finally {
    setLanguage(original);
  }
});

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
    jokers: [{ kind: 'showRunners', available: false, uses: [{ usedAt: at, positions: [
      { ...position, playerId: caught.playerId }, { ...position, playerId: state.players[2].playerId },
    ] }] }],
  };
  assert.deepEqual(revealMarkers(hunters), []);
  assert.deepEqual(gameHighlights(hunters), []);
  assert.deepEqual(playerMarkers(hunters).map((marker) => marker.id), [state.players[2].playerId]);
  assert.deepEqual(jokerMarkers(hunters).map((marker) => marker.title), ['Cy']);
  assert.deepEqual(gameMarkers(hunters).map((marker) => marker.id), [
    `joker-showRunners-0-${state.players[2].playerId}`, state.players[2].playerId, `caught-${caught.playerId}`,
  ]);
  const runners: GameStateRunnerResponse = {
    ...state, players, jokers: [{ kind: 'fakePosition', available: false, uses: [{ usedAt: at, usedBy: caught.playerId, point: position }] }],
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

test('spectators see real reports of runners and every hunter team with separate colors and frozen outcomes', () => {
  const report = { latitude: 52.51, longitude: 13.41, accuracyMeters: 5, recordedAt: '2026-11-08T12:00:58Z' };
  const endedAt = '2026-11-08T12:01:00Z';
  const spectator: GameStateRunnerResponse = {
    ...state, role: 'spectator', status: 'running',
    players: [
      { ...state.players[0], position: report, runner: { status: 'finished', endedAt } },
      { ...state.players[1], position: report, hunterTeam: 'Wolves' },
      { ...state.players[1], playerId: 'd'.repeat(32), name: 'Dan', position: { ...report, latitude: 52.55 }, hunterTeam: 'Foxes' },
      { ...state.players[2], position: { ...report, latitude: 52.56 } },
      { ...state.players[0], playerId: 'e'.repeat(32), name: 'Eve', position: report, runner: { status: 'caught', endedAt, caughtPosition: report } },
      { ...state.players[2], playerId: 'f'.repeat(32), name: 'No report' },
    ],
    // Historical snapshots must not be mixed into the real feed, even if stale data remains.
    jokers: [{ kind: 'showHunters', available: false, uses: [{ usedAt: endedAt, usedBy: state.playerId, positions: [{ ...report, playerId: state.players[1].playerId }] }] }],
  };
  const markers = spectatorMarkers(spectator);
  assert.deepEqual(gameMarkers(spectator), [...routeMarkers(state.route), ...markers]);
  assert.equal(markers.length, 5);
  assert.equal(new Set(markers.map((marker) => marker.id)).size, 5);
  assert.deepEqual(markers.map((marker) => marker.color), [SPECTATOR_COLORS.runner, SPECTATOR_COLORS.hunter, SPECTATOR_COLORS.hunter, SPECTATOR_COLORS.runner, MARKER_COLORS.caught]);
  assert.equal(SPECTATOR_COLORS.runner, '#2f80ed');
  assert.equal(SPECTATOR_COLORS.hunter, '#eb5757');
  assert.equal(markers[0].id, spectator.playerId); // Own frozen finish report is not filtered out.
  assert.equal(markers[0].label, '✓');
  assert.match(markers[0].title, /Ada.*you.*Runner.*Finished/);
  assert.match(markers[0].description, /before finishing.*no longer tracked/);
  assert.match(markers[1].title, /Bob.*Hunter/);
  assert.match(markers[1].description, /Wolves/);
  assert.match(markers[2].description, /Foxes/);
  assert.match(markers[3].title, /Cy.*Runner/);
  assert.match(markers[3].description, /Latest report/);
  assert.deepEqual(markers[3].coordinate, { latitude: 52.56, longitude: 13.41 });
  assert.equal(markers[4].label, '×');
  assert.match(markers[4].title, /Eve.*Caught/);
  assert.match(markers[4].description, /±5 m/);
  assert.deepEqual(gameHighlights(spectator), routeHighlights(state.playground, state.route)); // Route only, no joker/zone overlays.
  const expired = { ...spectator, players: spectator.players.map((player) => ({ ...player, position: undefined, runner: player.runner ? { ...player.runner, caughtPosition: undefined } : undefined })) };
  assert.deepEqual(gameMarkers(expired), routeMarkers(state.route)); // Landmarks remain without inventing reports.
});

test('spectator player reports never render before start or after end, or without the explicit role', () => {
  const report = { latitude: 52.51, longitude: 13.41, accuracyMeters: 5, recordedAt: state.serverTime };
  const viewer: GameStateRunnerResponse = {
    ...state, role: 'spectator', players: [{ ...state.players[0], runner: { status: 'caught', endedAt: state.serverTime, caughtPosition: report } }],
  };
  for (const status of ['lobby', 'countdown', 'ended'] as const) {
    assert.deepEqual(gameMarkers({ ...viewer, status }), routeMarkers(state.route));
    assert.deepEqual(gameHighlights({ ...viewer, status }), routeHighlights(state.playground, state.route));
  }
  assert.deepEqual(spectatorMarkers({ ...viewer, role: 'player', status: 'running' }), []);
  assert.equal(gameMarkers({ ...viewer, status: 'running' })[2].id, `caught-${viewer.playerId}`);
});

test('spectator permission disables location and finish eligibility and cannot be undone by stale responses', () => {
  const running: GameStateRunnerResponse = { ...state, status: 'running', startPositionCheckDisabled: true };
  // The explicit role is authoritative even if a stale player entry still says active.
  const spectator: GameStateRunnerResponse = { ...running, role: 'spectator' };
  assert.equal(isSpectator(undefined), false);
  assert.equal(isSpectator(running), false);
  assert.equal(isSpectator(spectator), true);
  assert.equal(canShareGameLocation(spectator), false);
  assert.equal(canFinish(spectator, 0, 0), false);
  assert.equal(shouldAcceptGameState(running, spectator), true);
  assert.equal(shouldAcceptGameState(spectator, { ...running, serverTime: '2026-11-08T13:00:00Z' }), false);
  assert.equal(shouldAcceptGameState(spectator, { ...spectator, serverTime: '2026-11-08T13:00:00Z' }), true);
  const ended = { ...spectator, status: 'ended' as const };
  assert.equal(shouldAcceptGameState(ended, spectator), false);
  assert.deepEqual(gameDestination(ended), gameResultRoute(spectator.sessionId));
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
  const runners: GameStateRunnerResponse = { ...state, players, jokers: [{ kind: 'fakePosition', available: false, uses: [{ usedAt: at, usedBy: finished.playerId, point: position }] }] };
  const hunters: GameStateHunterResponse = { ...hunterState, players, jokers: [{ kind: 'showRunners', available: false, uses: [{ usedAt: at, positions: [{ ...position, playerId: finished.playerId }] }] }] };
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

test('a used joker becomes one marker per runner it showed, for every use', () => {
  const position = (playerId: string, latitude: number) => ({ playerId, latitude, longitude: 13.4, accuracyMeters: 5, recordedAt: '2026-11-08T12:06:58Z' });
  const markers = jokerMarkers({
    ...hunterState,
    jokers: [{ kind: 'showRunners', available: true, uses: [
      { usedAt: '2026-11-08T12:07:00Z', positions: [position(state.players[0].playerId, 52.5), position(state.players[2].playerId, 52.6)] },
      { usedAt: '2026-11-08T12:37:00Z', positions: [position(state.players[0].playerId, 52.55)] },
    ] }],
  });
  assert.deepEqual(markers.map((m) => [m.title, m.coordinate.latitude, m.color]), [
    ['Ada', 52.5, MARKER_COLORS.joker],
    ['Cy', 52.6, MARKER_COLORS.joker],
    ['Ada', 52.55, MARKER_COLORS.joker],
  ]);
  assert.equal(new Set(markers.map((m) => m.id)).size, 3); // Later snapshots keep earlier ones.
  assert.match(markers[0].description, /Joker/);
  assert.deepEqual(jokerMarkers({ ...hunterState, jokers: [{ kind: 'showRunners', available: true, uses: [] }] }), []);
});

test('every queried zone is highlighted and marked in its center, colored by its answer', () => {
  const jokers = [{ kind: 'queryZone' as const, available: false, uses: [
    { usedAt: '2026-11-08T12:07:00Z', cell: 'A1', found: true },
    { usedAt: '2026-11-08T12:08:00Z', cell: 'F6', found: false },
    { usedAt: '2026-11-08T12:09:00Z', cell: 'A1', found: false },
  ] }];
  const zones = jokerZones({ ...hunterState, jokers });
  assert.deepEqual(zones.map((z) => [z.cell, z.color]), [['A1', ZONE_COLORS.found], ['F6', ZONE_COLORS.empty], ['A1', ZONE_COLORS.empty]]);
  assert.match(zones[0].title, /at least one runner/);
  assert.match(zones[1].title, /no runner/);
  const markers = jokerMarkers({ ...hunterState, jokers });
  const b = cellBounds(state.playground, 'A1');
  assert.deepEqual(markers[0].coordinate, { latitude: (b.north + b.south) / 2, longitude: (b.east + b.west) / 2 });
  assert.equal(markers[0].color, ZONE_COLORS.found);
  assert.equal(new Set(markers.map((m) => m.id)).size, 3); // The same cell queried twice keeps both answers.
  assert.deepEqual(jokerZones({ ...hunterState, jokers: [{ kind: 'queryZone', available: true, uses: [] }] }), []);
});

test('each personal fake position is a marker for its runner', () => {
  const points = [{ latitude: 52.53, longitude: 13.41 }, { latitude: 52.54, longitude: 13.42 }];
  const markers = fakePositionMarkers({
    ...state,
    jokers: [{ kind: 'fakePosition', available: false, uses: points.map((point, i) => ({ usedAt: `2026-11-08T12:0${7 + i}:00Z`, usedBy: state.players[0].playerId, point })) }],
  });
  assert.deepEqual(markers.map((m) => m.coordinate), points);
  assert.equal(new Set(markers.map((m) => m.id)).size, 2);
  assert.equal(markers[0].color, MARKER_COLORS.joker);
  assert.match(markers[0].title, /Ada/);
  assert.deepEqual(fakePositionMarkers({ ...state, jokers: [{ kind: 'fakePosition', available: true, uses: [] }] }), []);
});

test('the shown hunters are markers in their own color for every runner and use', () => {
  const position = { playerId: state.players[1].playerId, latitude: 52.51, longitude: 13.4, accuracyMeters: 5, recordedAt: '2026-11-08T12:06:58Z' };
  const running = {
    ...state,
    jokers: [{ kind: 'showHunters' as const, available: false, uses: [
      { usedAt: '2026-11-08T12:07:00Z', usedBy: state.players[0].playerId, positions: [position] },
      { usedAt: '2026-11-08T12:37:00Z', usedBy: state.players[2].playerId, positions: [{ ...position, latitude: 52.55 }] },
    ] }],
  };
  assert.deepEqual(shownHunterMarkers(running).map((m) => [m.title, m.coordinate.latitude, m.color]), [['Bob', 52.51, MARKER_COLORS.shownHunter], ['Bob', 52.55, MARKER_COLORS.shownHunter]]);
  assert.notEqual(MARKER_COLORS.shownHunter, MARKER_COLORS.joker);
  assert.ok(gameMarkers(running).some((m) => m.id === `joker-showHunters-1-${position.playerId}`));
  assert.deepEqual(shownHunterMarkers({ ...state, jokers: [{ kind: 'showHunters', available: true, uses: [] }] }), []);
});

test('zone alerts name the hunter and team from the players and stay in the top card for five minutes', () => {
  const bob = { ...state.players[1], hunterTeam: 'Wolves' };
  const alerted = {
    ...state,
    players: [state.players[0], bob, state.players[2]],
    zoneAlerts: [{ at: '2026-11-08T12:20:00Z', hunterId: bob.playerId, cell: 'C4' }],
  };
  const [entry] = zoneAlertEntries(alerted);
  assert.equal(entry.hunterId, bob.playerId);
  assert.match(entry.text, /Bob \(Wolves\) was in your zone C4$/);
  const at = Date.parse('2026-11-08T12:20:00Z');
  assert.equal(recentZoneAlert(alerted, at + ZONE_ALERT_HINT_SECONDS * 1000), entry.text);
  assert.equal(recentZoneAlert(alerted, at + ZONE_ALERT_HINT_SECONDS * 1000 + 1), null);
  assert.equal(recentZoneAlert(state, at), null);
  assert.deepEqual(zoneAlertEntries(hunterState), []);
});

test('the compass becomes one cell-long arrow per direction from the origin, without names', () => {
  const origin = { latitude: 52.52, longitude: 13.405 };
  const running = {
    ...hunterState,
    jokers: [{ kind: 'compass' as const, available: false, uses: [
      { usedAt: '2026-11-08T12:07:00Z', origin, bearings: [0, 90] },
      { usedAt: '2026-11-08T12:37:00Z', origin: { latitude: 52.5, longitude: 13.4 }, bearings: [180] },
    ] }],
  };
  const [north, east, south, ...rest] = compassLines(running);
  assert.equal(rest.length, 0);
  assert.deepEqual(north.coordinates[0], origin);
  assert.equal(north.color, MARKER_COLORS.joker);
  // The tip lies one cell away in the given direction.
  const tipNorth = north.coordinates[1];
  assert.ok(Math.abs(distanceMeters(origin, tipNorth) - state.playground.cellSizeMeters) < 1);
  assert.ok(tipNorth.latitude > origin.latitude && Math.abs(tipNorth.longitude - origin.longitude) < 1e-9);
  const tipEast = east.coordinates[1];
  assert.ok(tipEast.longitude > origin.longitude && Math.abs(tipEast.latitude - origin.latitude) < 1e-9);
  assert.ok(south.coordinates[1].latitude < 52.5);
  // Shaft, then both barbs from the tip.
  assert.equal(north.coordinates.length, 5);
  assert.deepEqual(north.coordinates[3], tipNorth);
  assert.equal(new Set([north, east, south].map((line) => line.id)).size, 3);
  const markers = compassMarkers(running);
  assert.deepEqual(markers.map((m) => m.coordinate), [origin, { latitude: 52.5, longitude: 13.4 }]);
  assert.ok(gameMarkers(running).some((m) => m.id === 'joker-compass-1'));
  assert.deepEqual(compassLines({ ...hunterState, jokers: [{ kind: 'compass', available: true, uses: [] }] }), []);
});

test('every trap is a circle of the trigger radius, colored waiting, active, then triggered, and no pin', () => {
  const point = { latitude: 52.52, longitude: 13.405 };
  const trap = { usedAt: '2026-11-08T12:00:00Z', point, activeAt: '2026-11-08T12:20:00Z' };
  const placed = { ...hunterState, jokers: [{ kind: 'trap' as const, available: false, uses: [trap] }] };
  const at = (iso: string) => Date.parse(iso);
  assert.deepEqual(trapCircles(placed, at('2026-11-08T12:19:59Z')), [{ id: 'joker-trap-0', center: point, radius: TRAP_RADIUS_METERS, color: TRAP_COLORS.waiting }]);
  assert.equal(trapCircles(placed, at('2026-11-08T12:20:00Z'))[0].color, TRAP_COLORS.active);
  const other = { usedAt: '2026-11-08T12:10:00Z', point: { latitude: 52.53, longitude: 13.41 }, activeAt: '2026-11-08T12:30:00Z' };
  const both = { ...hunterState, jokers: [{ kind: 'trap' as const, available: true, uses: [{ ...trap, triggeredAt: '2026-11-08T12:31:00Z' }, other] }] };
  assert.deepEqual(trapCircles(both, at('2026-11-08T12:31:00Z')).map((c) => [c.id, c.color]), [['joker-trap-0', TRAP_COLORS.triggered], ['joker-trap-1', TRAP_COLORS.active]]);
  assert.ok(!gameMarkers(both).some((m) => m.id.includes('trap')));
  assert.deepEqual(trapCircles({ ...hunterState, jokers: [{ kind: 'trap', available: true, uses: [] }] }, at('2026-11-08T12:31:00Z')), []);
  assert.equal(serverNow({ ...hunterState, serverTime: '2026-11-08T12:00:00Z' }, 1000, 3500), at('2026-11-08T12:00:02.5Z'));
});

test('remaining uses are the allowance minus the uses, never negative', () => {
  const settings = { ...state.jokerSettings, showRunners: { allowance: 3 }, queryZone: { allowance: 0 } };
  const snapshot = { ...hunterState, jokerSettings: settings };
  const use = { usedAt: '2026-11-08T12:07:00Z', positions: [] };
  assert.equal(remainingUses(snapshot, { kind: 'showRunners', available: true, uses: [use] }), 2);
  assert.equal(remainingUses(snapshot, { kind: 'showRunners', available: false, uses: [use, use, use, use] }), 0);
  assert.equal(remainingUses(snapshot, { kind: 'queryZone', available: false, uses: [] }), 0);
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
