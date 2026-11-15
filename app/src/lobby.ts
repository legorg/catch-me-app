import type { GameStateResponse, PlayerResponse } from './api/types';
import { getLanguage, translate } from './i18n';
import { cellBounds } from './playground';

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
 * up to the newest, which the server sends last.
 */
export function revealMarkers(state: GameStateResponse): (PlayerMarker & { label: string })[] {
  return state.players.flatMap((p) => (p.reveals ?? []).map((reveal, index) => ({
    id: `reveal-${p.playerId}-${reveal.revealedAt}`,
    coordinate: { latitude: reveal.latitude, longitude: reveal.longitude },
    title: p.name,
    description: translate('marker.revealedAt', { time: clockTime(reveal.revealedAt, true) }),
    color: MARKER_COLORS.revealed,
    label: String(index + 1),
  })));
}

/** Markers for what the team's used jokers showed: each runner where they were when it was used, and the queried zone with its answer. */
export function jokerMarkers(state: GameStateResponse): PlayerMarker[] {
  return state.jokers.flatMap((joker) => [
    ...(joker.positions ?? []).map((position) => ({
      id: `joker-${joker.kind}-${position.playerId}`,
      coordinate: { latitude: position.latitude, longitude: position.longitude },
      title: state.players.find((p) => p.playerId === position.playerId)?.name ?? '',
      description: translate('joker.marker', { time: clockTime(joker.usedAt ?? position.recordedAt, true) }),
      color: MARKER_COLORS.joker,
    })),
    ...queriedZones([joker]).map((zone) => {
      const b = cellBounds(state.playground, zone.cell);
      return {
        id: `joker-${joker.kind}-${zone.cell}`,
        coordinate: { latitude: (b.north + b.south) / 2, longitude: (b.east + b.west) / 2 },
        title: zone.title,
        description: translate('joker.marker', { time: clockTime(joker.usedAt ?? '', true) }),
        color: zone.color,
      };
    }),
  ]);
}

/** The cells the team queried, colored by the answer: green with a runner inside, red without. */
export function jokerZones(state: GameStateResponse): { cell: string; title: string; color: string }[] {
  return queriedZones(state.jokers);
}

function queriedZones(jokers: GameStateResponse['jokers']) {
  return jokers.flatMap((joker) => joker.cell && joker.found !== undefined ? [{
    cell: joker.cell,
    title: translate(joker.found ? 'joker.zoneFound' : 'joker.zoneEmpty', { cell: joker.cell }),
    color: joker.found ? ZONE_COLORS.found : ZONE_COLORS.empty,
  }] : []);
}

export const ZONE_COLORS = { found: '#27ae60', empty: '#eb5757' } as const;

export const SKETCH_COLOR = '#9b51e0';

/** At most `max` points (at least 2), evenly spread over the line and always keeping its first and last point. */
export function thinPoints<T>(points: T[], max: number): T[] {
  if (points.length <= max) return points;
  return Array.from({ length: max }, (_, i) => points[Math.round((i * (points.length - 1)) / (max - 1))]);
}

/** The lines the player's hunter team drew; the server sends them only to that team. */
export function sketchLines(state: GameStateResponse): { id: string; coordinates: { latitude: number; longitude: number }[]; color: string }[] {
  return state.sketches.map((sketch) => ({ id: sketch.sketchId, coordinates: sketch.points, color: SKETCH_COLOR }));
}

function clockTime(iso: string, seconds = false): string {
  return new Date(iso).toLocaleTimeString(getLanguage(), { hour: '2-digit', minute: '2-digit', second: seconds ? '2-digit' : undefined });
}
