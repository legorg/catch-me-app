import { isGameStateHunterResponse, isGameStateRunnerResponse, type GameRoute, type GameStateHunterResponse, type GameStateResponse, type GameStateRunnerResponse, type JoinCodes, type PlayerResponse, type Playground, type RevealResponse } from './api/types';
import { getLanguage, translate } from './i18n';
import { isActivePlayer, isSpectator } from './game-state';
import { cellAt, cellBounds, offsetPoint, START_RADIUS_METERS } from './playground';

/**
 * Whole seconds until a server timestamp, or null without one.
 * Uses the server's clock (serverTime) plus the time elapsed locally since the
 * response arrived, so a wrong phone clock does not matter.
 */
export function secondsUntil(target: string | undefined, state: GameStateResponse, receivedAt: number, now: number): number | null {
  if (!target) return null;
  const remaining = Date.parse(target) - Date.parse(state.serverTime) - (now - receivedAt);
  return Math.max(0, Math.ceil(remaining / 1000));
}

export function secondsUntilStart(state: GameStateResponse, receivedAt: number, now: number): number | null {
  return secondsUntil(state.startsAt, state, receivedAt, now);
}

/** For callbacks that must be passed but have nothing to do. */
export function noop() {}

/** The lobby's explanation of how the game starts (or, once counting down, what is happening). */
export function lobbyInfo(counting: boolean, testMode: boolean): string {
  if (counting) return translate(testMode ? 'lobby.countdownInfoTest' : 'lobby.countdownInfo');
  return translate(testMode ? 'lobby.startInfoTest' : 'lobby.startInfo');
}

/** Where the runner stands relative to the start cell check. */
export function startCheckMessage(testMode: boolean, atStart: boolean, metersToStart: number | null): string {
  if (testMode) return translate('lobby.checkDisabled');
  if (atStart) return translate('lobby.atStart');
  if (metersToStart !== null) return translate('lobby.goToStart', { meters: Math.round(metersToStart), radius: START_RADIUS_METERS });
  return translate('lobby.waitingForLocation');
}

export function formatDuration(seconds: number): string {
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`;
}

export function playersByTeam(players: PlayerResponse[]) {
  return {
    runner: players.filter((p) => p.team === 'runner'),
    hunter: players.filter((p) => p.team === 'hunter'),
    spectator: players.filter((p) => p.team === 'spectator'),
  };
}

export const TEAM_LABELS = { runner: 'team.runners', hunter: 'team.hunters', spectator: 'team.spectators' } as const;

/** Share just the selected invitation, never other codes, game IDs or credentials. */
export function joinInvitationMessage(codes: JoinCodes, team: keyof JoinCodes): string {
  return translate('joinCodes.invitation', { team: translate(TEAM_LABELS[team]), code: codes[team] });
}

/** Existing hunter team names, in the order their first member joined. */
export function hunterTeams(players: PlayerResponse[]): string[] {
  return [...new Set(players.flatMap((p) => (p.hunterTeam ? [p.hunterTeam] : [])))];
}

export const MARKER_COLORS = { me: '#27ae60', teammate: '#2f80ed', revealed: '#eb5757', joker: '#f2994a', shownHunter: '#6d4c41', caught: '#000000' } as const;

export const SPECTATOR_COLORS = { runner: '#2f80ed', hunter: '#eb5757' } as const;

export type PlayerMarker = {
  id: string;
  coordinate: { latitude: number; longitude: number };
  title: string;
  description: string;
  color: string;
  label?: string;
};

/** Only the authenticated spectator feed, never reconstructed from reveals or jokers. */
export function spectatorMarkers(state: GameStateResponse): PlayerMarker[] {
  if (!isSpectator(state) || state.status !== 'running') return [];
  return [
    ...state.players.flatMap((player) => {
      if (player.team === 'spectator' || !player.position || player.runner?.status === 'caught') return [];
      const { latitude, longitude, accuracyMeters, recordedAt } = player.position;
      const finished = player.runner?.status === 'finished';
      const name = player.playerId === state.playerId ? translate('common.you', { name: player.name }) : player.name;
      return [{
        id: player.playerId,
        coordinate: { latitude, longitude },
        title: translate(finished ? 'spectator.finishedMarker' : player.team === 'runner' ? 'spectator.runnerMarker' : 'spectator.hunterMarker', { name }),
        description: [
          translate(finished ? 'spectator.frozenReport' : 'spectator.latestReport'),
          player.hunterTeam ? translate('lobby.hunterTeam', { name: player.hunterTeam }) : '',
          translate('marker.accuracy', { meters: Math.round(accuracyMeters), time: clockTime(recordedAt, true) }),
        ].filter(Boolean).join(' · '),
        color: SPECTATOR_COLORS[player.team],
        ...(finished ? { label: '✓' } : {}),
      }];
    }),
    ...caughtMarkers(state),
  ];
}

/** Map markers for every live position the server allowed this player to see. */
export function playerMarkers(state: GameStateResponse): PlayerMarker[] {
  return state.players.flatMap((p) => {
    if (!p.position || !isActivePlayer(p)) return [];
    const { latitude, longitude, accuracyMeters, recordedAt } = p.position;
    const me = p.playerId === state.playerId;
    return [{
      id: p.playerId,
      coordinate: { latitude, longitude },
      title: me ? translate('common.you', { name: p.name }) : p.name,
      description: translate('marker.accuracy', { meters: Math.round(accuracyMeters), time: clockTime(recordedAt) }),
      color: me ? MARKER_COLORS.me : MARKER_COLORS.teammate,
    }];
  });
}

/** The last real report at capture, visible to everyone, including the caught runner. */
export function caughtMarkers(state: GameStateResponse): (PlayerMarker & { label: string })[] {
  return state.players.flatMap((player) => player.runner?.status === 'caught' && player.runner.endedAt && player.runner.caughtPosition ? [{
    id: `caught-${player.playerId}`,
    coordinate: { latitude: player.runner.caughtPosition.latitude, longitude: player.runner.caughtPosition.longitude },
    title: translate('catch.marker', { name: player.name }),
    description: translate('catch.markerTime', { time: clockTime(player.runner.endedAt, true) })
      + (isSpectator(state) ? ` · ${translate('marker.accuracy', { meters: Math.round(player.runner.caughtPosition.accuracyMeters), time: clockTime(player.runner.caughtPosition.recordedAt, true) })}` : ''),
    color: MARKER_COLORS.caught,
    label: '×',
  }] : []);
}

/**
 * Markers for every reveal the server sent (hunters only), numbered per runner from 1 (the oldest)
 * up to the newest, which the server sends last. A zone reveal sits in the middle of its cell;
 * one outside the playground has no marker but keeps its number.
 */
export function revealMarkers(state: GameStateResponse): (PlayerMarker & { label: string })[] {
  return state.players.flatMap((p) => (isActivePlayer(p) ? p.reveals ?? [] : []).flatMap((reveal, index) => {
    const coordinate = revealCoordinate(state.playground, reveal);
    if (!coordinate) return [];
    const time = clockTime(reveal.revealedAt, true);
    return [{
      id: `reveal-${p.playerId}-${reveal.revealedAt}`,
      coordinate,
      title: p.name,
      description: reveal.kind === 'zone' ? translate('marker.zoneRevealedAt', { cell: reveal.cell, time }) : translate('marker.revealedAt', { time }),
      color: MARKER_COLORS.revealed,
      label: String(index + 1),
    }];
  }));
}

/** Where a reveal's marker goes: at the position, in the middle of the zone, or nowhere. */
function revealCoordinate(playground: Playground, reveal: RevealResponse): { latitude: number; longitude: number } | null {
  switch (reveal.kind) {
    case 'position':
      return { latitude: reveal.position.latitude, longitude: reveal.position.longitude };
    case 'zone':
      return cellCenter(playground, reveal.cell);
    default:
      return null;
  }
}

/** The cells of zone reveals (runner joker), where hunters saw only the cell and not the position. */
export function revealZones(state: GameStateResponse): { cell: string; title: string; color: string }[] {
  return state.players.flatMap((p) => (isActivePlayer(p) ? p.reveals ?? [] : []).flatMap((reveal) => reveal.kind === 'zone'
    ? [{ cell: reveal.cell, title: p.name, color: ZONE_COLORS.revealed }]
    : []));
}

function cellCenter(playground: Playground, cell: string) {
  const b = cellBounds(playground, cell);
  return { latitude: (b.north + b.south) / 2, longitude: (b.east + b.west) / 2 };
}

/** Markers for what the team's used jokers showed: each runner where they were when it was used, and the queried zone with its answer. */
export function jokerMarkers(state: GameStateHunterResponse): PlayerMarker[] {
  return state.jokers.flatMap((joker) => [
    ...(joker.kind === 'showRunners' ? joker.positions ?? [] : []).filter((position) =>
      isActivePlayer(state.players.find((player) => player.playerId === position.playerId))).map((position) => ({
      id: `joker-${joker.kind}-${position.playerId}`,
      coordinate: { latitude: position.latitude, longitude: position.longitude },
      title: state.players.find((p) => p.playerId === position.playerId)?.name ?? '',
      description: translate('joker.marker', { time: clockTime(joker.usedAt ?? position.recordedAt, true) }),
      color: MARKER_COLORS.joker,
    })),
    ...queriedZones([joker]).map((zone) => ({
      id: `joker-${joker.kind}-${zone.cell}`,
      coordinate: cellCenter(state.playground, zone.cell),
      title: zone.title,
      description: translate('joker.marker', { time: clockTime(joker.usedAt ?? '', true) }),
      color: zone.color,
    })),
  ]);
}

/** The cells the team queried, colored by the answer: green with a runner inside, red without. */
export function jokerZones(state: GameStateHunterResponse): { cell: string; title: string; color: string }[] {
  return queriedZones(state.jokers);
}

function queriedZones(jokers: GameStateHunterResponse['jokers']) {
  return jokers.flatMap((joker) => joker.kind === 'queryZone' && joker.cell && joker.found !== undefined ? [{
    cell: joker.cell,
    title: translate(joker.found ? 'joker.zoneFound' : 'joker.zoneEmpty', { cell: joker.cell }),
    color: joker.found ? ZONE_COLORS.found : ZONE_COLORS.empty,
  }] : []);
}

export const ZONE_COLORS = { found: '#27ae60', empty: '#eb5757', revealed: '#f2c94c' } as const;

export const SKETCH_COLOR = '#9b51e0';

/** At most `max` points (at least 2), evenly spread over the line and always keeping its first and last point. */
export function thinPoints<T>(points: T[], max: number): T[] {
  if (points.length <= max) return points;
  return Array.from({ length: max }, (_, i) => points[Math.round((i * (points.length - 1)) / (max - 1))]);
}

/** The lines the player's hunter team drew; the server sends them only to that team. */
export function sketchLines(state: GameStateHunterResponse): { id: string; coordinates: { latitude: number; longitude: number }[]; color: string }[] {
  return state.sketches.map((sketch) => ({ id: sketch.sketchId, coordinates: sketch.points, color: SKETCH_COLOR }));
}

/** Arrowhead barbs: their angle to the shaft and their length as a share of it. */
const ARROWHEAD_DEGREES = 25;
const ARROWHEAD_SHARE = 0.25;

/**
 * The compass joker's arrows: one per direction the server sent, from where the hunter stood, all one cell
 * long so they say nothing about the distance. Drawn as one line: the shaft to the tip, then both barbs.
 */
export function compassLines(state: GameStateHunterResponse): { id: string; coordinates: { latitude: number; longitude: number }[]; color: string }[] {
  const length = state.playground.cellSizeMeters;
  return state.jokers.flatMap((joker) => {
    if (joker.kind !== 'compass' || !joker.origin) return [];
    const origin = joker.origin;
    return (joker.bearings ?? []).map((bearing, index) => {
      const tip = offsetPoint(origin, bearing, length);
      const barb = (side: number) => offsetPoint(tip, bearing + 180 + side * ARROWHEAD_DEGREES, length * ARROWHEAD_SHARE);
      return { id: `joker-compass-${index}`, coordinates: [origin, tip, barb(-1), tip, barb(1)], color: MARKER_COLORS.joker };
    });
  });
}

/** Where the hunter stood when the team used the compass, the arrows' common start. */
export function compassMarkers(state: GameStateHunterResponse): PlayerMarker[] {
  return state.jokers.flatMap((joker) => joker.kind === 'compass' && joker.origin ? [{
    id: 'joker-compass',
    coordinate: joker.origin,
    title: translate('joker.compass.title'),
    description: translate('joker.marker', { time: clockTime(joker.usedAt ?? '', true) }),
    color: MARKER_COLORS.joker,
  }] : []);
}

/** How close a runner must come to trigger a trap; the server uses the same radius. */
export const TRAP_RADIUS_METERS = 100;

export const TRAP_COLORS = { waiting: '#828282', active: '#f2994a', triggered: '#eb5757' } as const;

export type TrapStatus = keyof typeof TRAP_COLORS;

type HunterJoker = GameStateHunterResponse['jokers'][number];

/** Whether a placed trap still waits for its activation, is active, or was triggered, at the server time serverNow (ms). */
export function trapStatus(joker: Extract<HunterJoker, { kind: 'trap' }>, serverNow: number): TrapStatus {
  if (joker.triggeredAt) return 'triggered';
  return joker.activeAt && Date.parse(joker.activeAt) <= serverNow ? 'active' : 'waiting';
}

/** The team's placed trap as a circle of the trigger radius, colored by its status. */
export function trapCircles(state: GameStateHunterResponse, serverNow: number): { id: string; center: { latitude: number; longitude: number }; radius: number; color: string }[] {
  return state.jokers.flatMap((joker) => joker.kind === 'trap' && joker.point ? [{
    id: 'joker-trap',
    center: joker.point,
    radius: TRAP_RADIUS_METERS,
    color: TRAP_COLORS[trapStatus(joker, serverNow)],
  }] : []);
}

/** The server's clock now: its time in the response plus the time elapsed locally since it arrived. */
export function serverNow(state: GameStateResponse, receivedAt: number, now: number): number {
  return Date.parse(state.serverTime) + (now - receivedAt);
}

/** Start and target colors on the map; the same values as colors.start / colors.target (styles.ts cannot be loaded by the node tests). */
export const ROUTE_COLORS = { start: '#2f80ed', target: '#eb5757' } as const;

/** The route's start and stop names, falling back to the generic labels. */
export function routeTitles(route: GameRoute): { start: string; stop: string } {
  return { start: route.start.title || translate('lobby.start'), stop: route.stop.title || translate('lobby.target') };
}

/** Why the runners cannot regenerate the route now (the server checks the same), or null if they can. */
export function routeChangeBlocker(state: GameStateRunnerResponse): 'notLobby' | 'noneLeft' | 'runnerReady' | null {
  if (state.status !== 'lobby') return 'notLobby';
  if (state.routeRegenerationsLeft <= 0) return 'noneLeft';
  if (state.players.some((p) => p.team === 'runner' && p.ready)) return 'runnerReady';
  return null;
}

/** The grid cells of the route's start and stop (each left out when it lies outside the playground). */
export function routeHighlights(playground: Playground, route: GameRoute): { cell: string; title: string; color: string }[] {
  const titles = routeTitles(route);
  const start = cellAt(playground, route.start.coordinates);
  const stop = cellAt(playground, route.stop.coordinates);
  return [
    ...(start ? [{ cell: start, title: titles.start, color: ROUTE_COLORS.start }] : []),
    ...(stop ? [{ cell: stop, title: titles.stop, color: ROUTE_COLORS.target }] : []),
  ];
}

/** Map markers for the route's start and stop. */
export function routeMarkers(route: GameRoute): PlayerMarker[] {
  const titles = routeTitles(route);
  return [
    { id: 'route-start', coordinate: route.start.coordinates, title: titles.start, description: translate('lobby.start'), color: ROUTE_COLORS.start },
    { id: 'route-stop', coordinate: route.stop.coordinates, title: titles.stop, description: translate('lobby.target'), color: ROUTE_COLORS.target },
  ];
}

/** The requesting runner's own fakePosition point, once used. */
export function fakePositionMarkers(state: GameStateRunnerResponse): PlayerMarker[] {
  return state.jokers.flatMap((joker) => joker.kind === 'fakePosition' && joker.point &&
    isActivePlayer(state.players.find((player) => player.playerId === joker.usedBy)) ? [{
    id: 'joker-fakePosition',
    coordinate: joker.point,
    title: translate('joker.fakePosition.marker', { name: state.players.find((p) => p.playerId === joker.usedBy)?.name ?? '' }),
    description: translate('joker.marker', { time: clockTime(joker.usedAt ?? '', true) }),
    color: MARKER_COLORS.joker,
  }] : []);
}

export function shownHunterMarkers(state: GameStateRunnerResponse): PlayerMarker[] {
  return state.jokers.flatMap((joker) => (joker.kind === 'showHunters' ? joker.positions ?? [] : []).map((position) => ({
    id: `joker-${joker.kind}-${position.playerId}`,
    coordinate: { latitude: position.latitude, longitude: position.longitude },
    title: state.players.find((p) => p.playerId === position.playerId)?.name ?? '',
    description: translate('joker.showHunters.marker', { time: clockTime(joker.usedAt ?? position.recordedAt, true) }),
    color: MARKER_COLORS.shownHunter,
  })));
}

/** Only runners and spectators receive the shared start and destination. */
export function visibleRoute(state: GameStateResponse): GameRoute | undefined {
  return state.team === 'hunter' ? undefined : state.route;
}

/** Route cells for runners/spectators; queried zones and zone-only reveals for active hunters. */
export function gameHighlights(state: GameStateResponse) {
  const route = visibleRoute(state);
  if (route) return routeHighlights(state.playground, route);
  if (isSpectator(state)) return [];
  if (isGameStateHunterResponse(state)) return [...jokerZones(state), ...revealZones(state)];
  return [];
}

/** Permitted markers, with catch pins last. The device supplies the active player's own live position. */
export function gameMarkers(state: GameStateResponse): PlayerMarker[] {
  if (isSpectator(state)) {
    const route = visibleRoute(state);
    return [...(route ? routeMarkers(route) : []), ...spectatorMarkers(state)];
  }
  return [
    ...(isGameStateRunnerResponse(state) ? [...routeMarkers(state.route), ...fakePositionMarkers(state), ...shownHunterMarkers(state)] : []),
    ...revealMarkers(state),
    ...(isGameStateHunterResponse(state) ? [...jokerMarkers(state), ...compassMarkers(state)] : []),
    ...playerMarkers(state).filter((marker) => marker.id !== state.playerId),
    ...caughtMarkers(state),
  ];
}

export function clockTime(iso: string, seconds = false): string {
  return new Date(iso).toLocaleTimeString(getLanguage(), { hour: '2-digit', minute: '2-digit', second: seconds ? '2-digit' : undefined });
}
