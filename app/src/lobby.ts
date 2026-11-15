import type { GameStateResponse, PlayerResponse } from './api/types';
import { getLanguage, translate } from './i18n';

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

export const MARKER_COLORS = { me: '#27ae60', teammate: '#2f80ed', revealed: '#eb5757' } as const;

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

const OLDEST_REVEAL_OPACITY = 0.15;
const REVEAL_FADE_STEP = 0.15;
const PREVIOUS_REVEAL_OPACITY = 0.7;

/**
 * Markers for every reveal the server sent (hunters only), oldest first from the server.
 * The newest reveal of a runner is opaque; each older one fades further.
 */
export function revealMarkers(state: GameStateResponse): (PlayerMarker & { opacity: number })[] {
  return state.players.flatMap((p) => {
    const reveals = p.reveals ?? [];
    return reveals.map((reveal, index) => {
      const age = reveals.length - 1 - index;
      return {
        id: `reveal-${p.playerId}-${reveal.revealedAt}`,
        coordinate: { latitude: reveal.latitude, longitude: reveal.longitude },
        title: p.name,
        description: translate('marker.revealedAt', { time: clockTime(reveal.revealedAt, true) }),
        color: MARKER_COLORS.revealed,
        opacity: age === 0 ? 1 : Math.max(OLDEST_REVEAL_OPACITY, PREVIOUS_REVEAL_OPACITY - (age - 1) * REVEAL_FADE_STEP),
      };
    });
  });
}

function clockTime(iso: string, seconds = false): string {
  return new Date(iso).toLocaleTimeString(getLanguage(), { hour: '2-digit', minute: '2-digit', second: seconds ? '2-digit' : undefined });
}
