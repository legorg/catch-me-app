import type { GameStateResponse } from './api/types';

export function ownPlayer(state: GameStateResponse | undefined) {
  return state?.players.find((player) => player.playerId === state.playerId);
}

export function isCaught(state: GameStateResponse | undefined): boolean {
  return !!ownPlayer(state)?.caughtAt;
}

export function canShareGameLocation(state: GameStateResponse | undefined): boolean {
  return state?.status === 'running' && !!ownPlayer(state) && !isCaught(state);
}

/** An older poll or in-flight position response must not undo a confirmed catch. */
export function shouldAcceptGameState(previous: GameStateResponse | undefined, incoming: GameStateResponse): boolean {
  if (!previous || previous.sessionId !== incoming.sessionId || previous.playerId !== incoming.playerId) return true;
  if (isCaught(previous) && !isCaught(incoming)) return false;
  return Date.parse(incoming.serverTime) >= Date.parse(previous.serverTime);
}
