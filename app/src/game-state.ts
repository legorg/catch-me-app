import type { GameStateResponse, PlayerResponse } from './api/types';

export function ownPlayer(state: GameStateResponse | undefined) {
  return state?.players.find((player) => player.playerId === state.playerId);
}

export function isCaught(state: GameStateResponse | undefined): boolean {
  return ownPlayer(state)?.runner?.status === 'caught';
}

export function isActivePlayer(player: PlayerResponse | undefined): boolean {
  return player?.team === 'hunter' || player?.runner?.status === 'active';
}

export function canShareGameLocation(state: GameStateResponse | undefined): boolean {
  return state?.status === 'running' && isActivePlayer(ownPlayer(state));
}

/** Late responses must not undo or change a confirmed runner outcome. */
export function shouldAcceptGameState(previous: GameStateResponse | undefined, incoming: GameStateResponse): boolean {
  if (!previous || previous.sessionId !== incoming.sessionId || previous.playerId !== incoming.playerId) return true;
  const runner = ownPlayer(previous)?.runner;
  if (runner && runner.status !== 'active' && ownPlayer(incoming)?.runner?.status !== runner.status) return false;
  return Date.parse(incoming.serverTime) >= Date.parse(previous.serverTime);
}
