import { isGameStateHunterResponse, isGameStateRunnerResponse, type GameRoute, type GameStateHunterResponse, type GameStateResponse, type GameStateRunnerResponse, type PlayerResponse, type Playground } from './api/types';
import { getLanguage, translate } from './i18n';
import { cellAt, cellBounds, START_RADIUS_METERS } from './playground';

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
  };
}

/** Existing hunter team names, in the order their first member joined. */
export function hunterTeams(players: PlayerResponse[]): string[] {
  return [...new Set(players.flatMap((p) => (p.hunterTeam ? [p.hunterTeam] : [])))];
}

export const MARKER_COLORS = { me: '#27ae60', teammate: '#2f80ed', revealed: '#eb5757', joker: '#f2994a' } as const;

export type PlayerMarker = {
  id: string;
  coordinate: { latitude: number; longitude: number };
  title: string;
  description: string;
  color: string;
};

/** Map markers for every live position the server allowed this player to see. */
export function playerMarkers(state: GameStateResponse): PlayerMarker[] {
  return state.players.flatMap((p) => {
    if (!p.position) return [];
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

/**
 * Markers for every reveal the server sent (hunters only), numbered per runner from 1 (the oldest)
 * up to the newest, which the server sends last. A zone-only reveal sits in the middle of its cell;
 * one outside the playground has no marker but keeps its number.
 */
export function revealMarkers(state: GameStateResponse): (PlayerMarker & { label: string })[] {
  return state.players.flatMap((p) => (p.reveals ?? []).flatMap(({ position, cell, revealedAt }, index) => {
    const coordinate = position ? { latitude: position.latitude, longitude: position.longitude } : cell ? cellCenter(state.playground, cell) : null;
    if (!coordinate) return [];
    const time = clockTime(revealedAt, true);
    return [{
      id: `reveal-${p.playerId}-${revealedAt}`,
      coordinate,
      title: p.name,
      description: position ? translate('marker.revealedAt', { time }) : translate('marker.zoneRevealedAt', { cell: cell ?? '', time }),
      color: MARKER_COLORS.revealed,
      label: String(index + 1),
    }];
  }));
}

/** The cells of zone-only reveals (runner joker), where hunters saw only the cell and not the position. */
export function revealZones(state: GameStateResponse): { cell: string; title: string; color: string }[] {
  return state.players.flatMap((p) => (p.reveals ?? []).flatMap((reveal) => reveal.zoneOnly && reveal.cell
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
    ...(joker.positions ?? []).map((position) => ({
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
  return jokers.flatMap((joker) => joker.cell && joker.found !== undefined ? [{
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

/** Every highlighted cell the player's team may see: the route (runners) or the queried zones and zone-only reveals (hunters). */
export function gameHighlights(state: GameStateResponse) {
  if (isGameStateRunnerResponse(state)) return routeHighlights(state.playground, state.route);
  if (isGameStateHunterResponse(state)) return [...jokerZones(state), ...revealZones(state)];
  return [];
}

/** Every marker the player's team may see, other players' live positions last (the player's own position is the device's). */
export function gameMarkers(state: GameStateResponse): PlayerMarker[] {
  return [
    ...(isGameStateRunnerResponse(state) ? routeMarkers(state.route) : []),
    ...revealMarkers(state),
    ...(isGameStateHunterResponse(state) ? jokerMarkers(state) : []),
    ...playerMarkers(state).filter((marker) => marker.id !== state.playerId),
  ];
}

function clockTime(iso: string, seconds = false): string {
  return new Date(iso).toLocaleTimeString(getLanguage(), { hour: '2-digit', minute: '2-digit', second: seconds ? '2-digit' : undefined });
}
