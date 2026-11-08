import type { GameStateResponse, PlayerResponse } from './api/generated/types.gen';

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

export const TEAM_LABELS = { runner: 'Runners', hunter: 'Hunters' } as const;

export const MARKER_COLORS = { me: '#27ae60', teammate: '#2f80ed', revealed: '#eb5757' } as const;

export type PlayerMarker = {
  id: string;
  coordinate: { latitude: number; longitude: number };
  title: string;
  description: string;
  color: string;
};

/** Map markers for every position the server allowed this player to see. */
export function playerMarkers(state: GameStateResponse): PlayerMarker[] {
  return state.players.flatMap((p) => {
    if (!p.position) return [];
    const { latitude, longitude, accuracyMeters, recordedAt, revealedAt } = p.position;
    const me = p.playerId === state.playerId;
    const time = (iso: string) => new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    return [{
      id: p.playerId,
      coordinate: { latitude, longitude },
      title: me ? `${p.name} (you)` : p.name,
      description: revealedAt ? `Revealed at ${time(revealedAt)}` : `±${Math.round(accuracyMeters)} m · ${time(recordedAt)}`,
      color: me ? MARKER_COLORS.me : revealedAt ? MARKER_COLORS.revealed : MARKER_COLORS.teammate,
    }];
  });
}
