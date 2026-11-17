import {
  activateJoker as postJoker,
  addSketch as postSketch,
  clearSketches as deleteSketches,
  createGame as postGame,
  getGameState as fetchGameState,
  getHealth,
  joinGame as postPlayer,
  leaveGame as deletePlayer,
  regenerateRoute as postRoute,
  setHunterTeam as putHunterTeam,
  setReady as putReady,
  updatePosition as putPosition,
} from './generated/sdk.gen';
import { client } from './generated/client.gen';
import type { Coordinates, CreateGameResponse, GameStateResponse, HealthResponse, JoinGameResponse, JokerRequest } from './types';
import { translate, type MessageKey } from '../i18n';
import {
  DEFAULT_CELL_SIZE_METERS, DEFAULT_GRID_SIZE, DEFAULT_REVEAL_INTERVAL_MINUTES, isValidCellSize, isValidGridSize, isValidRevealInterval,
} from '../playground';

export type { Coordinates, CreateGameResponse, GameStateResponse, HealthResponse, JoinCodes, JoinGameResponse, JokerRequest, JokerResponse } from './types';

export const API_URL = (
  process.env.EXPO_PUBLIC_API_URL || 'http://localhost:8080'
).trim().replace(/\/+$/, '');

// Development aid: shows in the Metro terminal which URL every request goes to (__DEV__ is unset under node tests).
if (typeof __DEV__ !== 'undefined' && __DEV__) {
  client.interceptors.request.use((request) => {
    console.log(`[api] ${request.method} ${request.url}`);
    return request;
  });
}

export const JOIN_CODE_PATTERN = /^[0-9]{6}$/;
export const MAX_NAME_LENGTH = 20;

/** An error response from the backend, with its HTTP status (e.g. 401 for an unknown player token). */
export class ApiError extends Error {
  constructor(message: string, readonly status: number) {
    super(message);
    this.name = 'ApiError';
  }
}

export function normalizeJoinCode(code: string): string {
  return code.trim();
}

export function isValidPlayerName(name: string): boolean {
  const length = [...name.trim()].length;
  return length >= 1 && length <= MAX_NAME_LENGTH;
}

export async function checkBackend(): Promise<HealthResponse> {
  return request((signal) => getHealth({ baseUrl: API_URL, signal }));
}

export async function createGame(
  center: { latitude: number; longitude: number },
  cellSizeMeters = DEFAULT_CELL_SIZE_METERS,
  gridSize = DEFAULT_GRID_SIZE,
  revealIntervalMinutes = DEFAULT_REVEAL_INTERVAL_MINUTES,
): Promise<CreateGameResponse> {
  if (!isValidCellSize(cellSizeMeters)) {
    throw new Error(translate('playground.cellSizeInvalid'));
  }
  if (!isValidGridSize(gridSize)) {
    throw new Error(translate('playground.gridSizeInvalid'));
  }
  if (!isValidRevealInterval(revealIntervalMinutes)) {
    throw new Error(translate('playground.revealIntervalInvalid'));
  }
  const body = { ...center, cellSizeMeters, gridSize, revealIntervalMinutes };
  return request((signal) => postGame({ baseUrl: API_URL, body, signal }), LANDMARK_TIMEOUT_MS, 'client.createTimeout');
}

export async function joinGame(code: string, name: string): Promise<JoinGameResponse> {
  const normalized = normalizeJoinCode(code);
  if (!JOIN_CODE_PATTERN.test(normalized)) {
    throw new Error(translate('client.joinCode'));
  }
  if (!isValidPlayerName(name)) {
    throw new Error(translate('client.name', { max: MAX_NAME_LENGTH }));
  }
  return request((signal) => postPlayer({ baseUrl: API_URL, body: { code: normalized, name: name.trim() }, signal }));
}

export async function getGameState(token: string): Promise<GameStateResponse> {
  return request((signal) => fetchGameState({ baseUrl: API_URL, auth: bearer(token), signal }));
}

export async function setReady(token: string, ready: boolean): Promise<GameStateResponse> {
  return request((signal) => putReady({ baseUrl: API_URL, auth: bearer(token), body: { ready }, signal }));
}

export async function leaveGame(token: string): Promise<void> {
  await request((signal) => deletePlayer({ baseUrl: API_URL, auth: bearer(token), signal }));
}

/** Gives the game a new route (runners, twice per game, while no runner is ready). */
export async function regenerateRoute(token: string): Promise<GameStateResponse> {
  return request((signal) => postRoute({ baseUrl: API_URL, auth: bearer(token), signal }), LANDMARK_TIMEOUT_MS, 'client.routeTimeout');
}

export async function setHunterTeam(token: string, name: string): Promise<GameStateResponse> {
  if (!isValidPlayerName(name)) {
    throw new Error(translate('client.teamName', { max: MAX_NAME_LENGTH }));
  }
  return request((signal) => putHunterTeam({ baseUrl: API_URL, auth: bearer(token), body: { name: name.trim() }, signal }));
}

export type JokerKind = JokerRequest['kind'];

/** Spends a joker of the player's team (once per game and team, while it runs), with exactly the fields that joker needs. */
export async function activateJoker(token: string, joker: JokerRequest): Promise<GameStateResponse> {
  return request((signal) => postJoker({ baseUrl: API_URL, auth: bearer(token), body: joker, signal }));
}

const MIN_SKETCH_POINTS = 2;
export const MAX_SKETCH_POINTS = 500;

/** Draws a line on the map of the player's hunter team (while the game runs). */
export async function addSketch(token: string, points: Coordinates[]): Promise<GameStateResponse> {
  if (points.length < MIN_SKETCH_POINTS || points.length > MAX_SKETCH_POINTS) {
    throw new Error(translate('client.sketchPoints', { min: MIN_SKETCH_POINTS, max: MAX_SKETCH_POINTS }));
  }
  const body = { points: points.map(({ latitude, longitude }) => ({ latitude, longitude })) };
  return request((signal) => postSketch({ baseUrl: API_URL, auth: bearer(token), body, signal }));
}

/** Removes all lines of the player's hunter team, whoever drew them. */
export async function clearSketches(token: string): Promise<GameStateResponse> {
  return request((signal) => deleteSketches({ baseUrl: API_URL, auth: bearer(token), signal }));
}

export async function updatePosition(
  token: string, position: { latitude: number; longitude: number; accuracyMeters: number },
): Promise<GameStateResponse> {
  return request((signal) => putPosition({ baseUrl: API_URL, auth: bearer(token), body: position, signal }));
}

// The API declares an apiKey scheme, so the generated client sends this value unchanged.
function bearer(token: string) {
  return `Bearer ${token}`;
}

type Result = { data?: unknown; error?: unknown; response?: Response };

const DEFAULT_TIMEOUT_MS = 5000;
// Requests that look up landmarks wait longer: the server gives up after 8 seconds
// (12 seconds write timeout), so 15 seconds leave room for a slow mobile network.
const LANDMARK_TIMEOUT_MS = 15_000;

// Runs a generated SDK call with a timeout and turns error
// responses into ApiError, keeping the backend's readable message.
async function request<R extends Result>(
  call: (signal: AbortSignal) => Promise<R>, timeoutMs = DEFAULT_TIMEOUT_MS, timeoutMessage: MessageKey = 'client.timeout',
): Promise<NonNullable<R['data']>> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const result = await call(controller.signal);
    // Network failures and aborts are returned, not thrown, by the generated client.
    if (result.error instanceof Error) {
      throw result.error;
    }
    const status = result.response?.status ?? 0;
    if (result.error !== undefined || status < 200 || status > 299) {
      const body = result.error as { error?: unknown } | null;
      const message = typeof body?.error === 'string'
        ? body.error
        : translate('client.status', { status });
      throw new ApiError(message, status);
    }
    return result.data as NonNullable<R['data']>;
  } catch (error) {
    console.log(error)
    if (controller.signal.aborted) {
      throw new Error(translate(timeoutMessage));
    }
    if (error instanceof Error) {
      throw error;
    }
    throw new Error(translate('client.unreachable'));
  } finally {
    clearTimeout(timeout);
  }
}
