import type { ControllerGameStateResponse } from './generated/types.gen';

// Readable names for the generated schema types.
export type {
  ControllerBounds as Bounds,
  ControllerCoordinates as Coordinates,
  ControllerCreateGameResponse as CreateGameResponse,
  ControllerGameResultResponse as GameResultResponse,
  ControllerGameRoute as GameRoute,
  ControllerGameStateHunterResponse as GameStateHunterResponse,
  ControllerGameStateResponse as GameStateResponse,
  ControllerGameStateRunnerResponse as GameStateRunnerResponse,
  ControllerGameStateSpectatorResponse as GameStateSpectatorResponse,
  ControllerHealthResponse as HealthResponse,
  ControllerJoinCodes as JoinCodes,
  ControllerJoinGameResponse as JoinGameResponse,
  ControllerJokerResponse as JokerResponse,
  ControllerJokerSettings as JokerSettings,
  ControllerPlayerResponse as PlayerResponse,
  ControllerPlayground as Playground,
  ControllerRevealResponse as RevealResponse,
  ControllerResultPlayerResponse as ResultPlayerResponse,
  ControllerRunnerStatus as RunnerStatus,
  ControllerTimelineEventResponse as TimelineEvent,
  ControllerTimelineResponse as TimelineResponse,
  ControllerUseJokerRequest as JokerRequest,
} from './generated/types.gen';

type GameState = ControllerGameStateResponse;

export function isGameStateRunnerResponse(state: GameState): state is Extract<GameState, { team: 'runner' }> {
  return state.team === 'runner';
}

export function isGameStateHunterResponse(state: GameState): state is Extract<GameState, { team: 'hunter' }> {
  return state.team === 'hunter';
}

export function isGameStateSpectatorResponse(state: GameState): state is Extract<GameState, { team: 'spectator' }> {
  return state.team === 'spectator';
}
