import type { GameStateResponse, PlayerResponse } from './api/generated/types.gen';

/**
 * Whole seconds until the scheduled start, or null without a countdown.
 * Uses the server's clock (serverTime) plus the time elapsed locally since the
 * response arrived, so a wrong phone clock does not matter.
 */
export function secondsUntilStart(state: GameStateResponse, receivedAt: number, now: number): number | null {
  if (!state.startsAt) return null;
  const remaining = Date.parse(state.startsAt) - Date.parse(state.serverTime) - (now - receivedAt);
  return Math.max(0, Math.ceil(remaining / 1000));
}

export function playersByTeam(players: PlayerResponse[]) {
  return {
    runner: players.filter((p) => p.team === 'runner'),
    hunter: players.filter((p) => p.team === 'hunter'),
  };
}

export const TEAM_LABELS = { runner: 'Runners', hunter: 'Hunters' } as const;
