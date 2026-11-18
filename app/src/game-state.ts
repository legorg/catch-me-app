import { isGameStateRunnerResponse, type GameStateResponse, type PlayerResponse } from './api/types';
import { distanceMeters, START_RADIUS_METERS } from './playground';

const LANDMARK_POSITION_MAX_AGE_MS = 30_000;

export function ownPlayer(state: GameStateResponse | undefined) {
  return state?.players.find((player) => player.playerId === state.playerId);
}

export function isActivePlayer(player: PlayerResponse | undefined): boolean {
  return player?.team === 'hunter' || player?.runner?.status === 'active';
}

export function canShareGameLocation(state: GameStateResponse | undefined): boolean {
  return state?.status === 'running' && isActivePlayer(ownPlayer(state));
}

/** UI eligibility only; the server repeats the check on the finish request. */
export function canFinish(state: GameStateResponse | undefined, receivedAt: number, now: number): boolean {
  if (!state || !isGameStateRunnerResponse(state) || state.status !== 'running' || !isActivePlayer(ownPlayer(state))) return false;
  if (state.startPositionCheckDisabled === true) return true;
  const position = ownPlayer(state)?.position;
  if (!position) return false;
  // Use server timestamps plus elapsed time since receipt, never the phone's absolute clock.
  const age = Date.parse(state.serverTime) + Math.max(0, now - receivedAt) - Date.parse(position.recordedAt);
  return age >= 0 && age <= LANDMARK_POSITION_MAX_AGE_MS
    && distanceMeters(position, state.route.stop.coordinates) <= START_RADIUS_METERS;
}

/** Late responses must not undo or change a confirmed runner outcome. */
export function shouldAcceptGameState(previous: GameStateResponse | undefined, incoming: GameStateResponse): boolean {
  if (!previous || previous.sessionId !== incoming.sessionId || previous.playerId !== incoming.playerId) return true;
  const runner = ownPlayer(previous)?.runner;
  if (runner && runner.status !== 'active' && ownPlayer(incoming)?.runner?.status !== runner.status) return false;
  return Date.parse(incoming.serverTime) >= Date.parse(previous.serverTime);
}
