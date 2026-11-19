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
  ControllerJokerKind as JokerKind,
  ControllerJokerResponse as JokerResponse,
  ControllerPlayerResponse as PlayerResponse,
  ControllerPlayground as Playground,
  ControllerPositionResponse as PositionResponse,
  ControllerRevealResponse as RevealResponse,
  ControllerResultPlayerResponse as ResultPlayerResponse,
  ControllerRunnerStatus as RunnerStatus,
  ControllerTimelineEventResponse as TimelineEvent,
  ControllerTimelinePlayer as TimelinePlayer,
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
