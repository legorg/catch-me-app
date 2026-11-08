import {
  createGame as postGame,
  getGameState as fetchGameState,
  getHealth,
  joinGame as postPlayer,
  leaveGame as deletePlayer,
  regenerateRoute as postRoute,
  setReady as putReady,
  updatePosition as putPosition,
} from './generated/sdk.gen';
import type { CreateGameResponse, GameStateResponse, HealthResponse, JoinGameResponse, Playground } from './generated/types.gen';
import { DEFAULT_CELL_SIZE_METERS, isValidCellSize } from '../playground';

export type {
  Coordinates, CreateGameResponse, GameStateResponse, HealthResponse, JoinGameResponse, PlayerResponse, Playground,
  PositionResponse, RouteResponse,
} from './generated/types.gen';

export const API_URL = (
  process.env.EXPO_PUBLIC_API_URL || 'http://localhost:8080'
).trim().replace(/\/+$/, '');

export const JOIN_CODE_PATTERN = /^[A-HJ-NP-Z2-9]{6}$/;
const CELL_PATTERN = /^[A-F][1-6]$/;
export const MAX_NAME_LENGTH = 20;

/** An error response from the backend, with its HTTP status (e.g. 401 for an unknown player token). */
export class ApiError extends Error {
  constructor(message: string, readonly status: number) {
    super(message);
    this.name = 'ApiError';
  }
}

export function normalizeJoinCode(code: string): string {
  return code.trim().toUpperCase();
}

export function isValidPlayerName(name: string): boolean {
  const length = [...name.trim()].length;
  return length >= 1 && length <= MAX_NAME_LENGTH;
}

export async function checkBackend(): Promise<HealthResponse> {
  const data = await request((signal) => getHealth({ baseUrl: API_URL, signal }));
  if (!isRecord(data) || data.status !== 'ok' || data.service !== 'catch-me-backend') {
    throw new Error('Unexpected backend response. Check the API URL.');
  }
  return data;
}

export async function createGame(center: { latitude: number; longitude: number }, cellSizeMeters = DEFAULT_CELL_SIZE_METERS): Promise<CreateGameResponse> {
  if (!isValidCellSize(cellSizeMeters)) {
    throw new Error('Cell size must be 0.1–5 km in steps of 0.1 km.');
  }
  const data = await request((signal) => postGame({ baseUrl: API_URL, body: { ...center, cellSizeMeters }, signal }));
  const game = isRecord(data) ? data.game : undefined;
  const codes = isRecord(data) ? data.joinCodes : undefined;
  if (!isRecord(game) || !isHex(game.sessionId, 32) || !isDate(game.createdAt) || !isPlayground(game.playground) ||
      !isRecord(codes) || !isJoinCode(codes.runner) || !isJoinCode(codes.hunter)) {
    throw new Error('Unexpected game response. Check the API URL.');
  }
  return data as CreateGameResponse;
}

export async function joinGame(code: string, name: string): Promise<JoinGameResponse> {
  const normalized = normalizeJoinCode(code);
  if (!JOIN_CODE_PATTERN.test(normalized)) {
    throw new Error('Enter the 6-character join code.');
  }
  if (!isValidPlayerName(name)) {
    throw new Error(`Enter a name with 1–${MAX_NAME_LENGTH} characters.`);
  }
  const data = await request((signal) => postPlayer({ baseUrl: API_URL, body: { code: normalized, name: name.trim() }, signal }));
  if (!isRecord(data) || !isHex(data.playerToken, 64)) {
    throw new Error('Unexpected join response. Check the API URL.');
  }
  assertGameState(data.state);
  return data as JoinGameResponse;
}

export async function getGameState(token: string): Promise<GameStateResponse> {
  const data = await request((signal) => fetchGameState({ baseUrl: API_URL, auth: bearer(token), signal }));
  assertGameState(data);
  return data;
}

export async function setReady(token: string, ready: boolean): Promise<GameStateResponse> {
  const data = await request((signal) => putReady({ baseUrl: API_URL, auth: bearer(token), body: { ready }, signal }));
  assertGameState(data);
  return data;
}

export async function leaveGame(token: string): Promise<void> {
  await request((signal) => deletePlayer({ baseUrl: API_URL, auth: bearer(token), signal }));
}

export async function regenerateRoute(token: string): Promise<GameStateResponse> {
  const data = await request((signal) => postRoute({ baseUrl: API_URL, auth: bearer(token), signal }));
  assertGameState(data);
  return data;
}

export async function updatePosition(
  token: string, position: { latitude: number; longitude: number; accuracyMeters: number },
): Promise<GameStateResponse> {
  const data = await request((signal) => putPosition({ baseUrl: API_URL, auth: bearer(token), body: position, signal }));
  assertGameState(data);
  return data;
}

// The API declares an apiKey scheme, so the generated client sends this value unchanged.
function bearer(token: string) {
  return `Bearer ${token}`;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function finite(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value);
}

function isHex(value: unknown, length: number): value is string {
  return typeof value === 'string' && value.length === length && /^[a-f0-9]+$/.test(value);
}

function isDate(value: unknown): value is string {
  return typeof value === 'string' && Number.isFinite(Date.parse(value));
}

function isJoinCode(value: unknown): value is string {
  return typeof value === 'string' && JOIN_CODE_PATTERN.test(value);
}

function isTeam(value: unknown): value is 'runner' | 'hunter' {
  return value === 'runner' || value === 'hunter';
}

function isRoute(value: unknown): boolean {
  return isRecord(value) && typeof value.start === 'string' && CELL_PATTERN.test(value.start) &&
    typeof value.end === 'string' && CELL_PATTERN.test(value.end) &&
    Number.isInteger(value.regenerationsLeft) && (value.regenerationsLeft as number) >= 0 && (value.regenerationsLeft as number) <= 2;
}

function isPosition(value: unknown): boolean {
  return isRecord(value) && finite(value.latitude) && Math.abs(value.latitude) <= 90 &&
    finite(value.longitude) && Math.abs(value.longitude) <= 180 &&
    finite(value.accuracyMeters) && value.accuracyMeters >= 0 && isDate(value.recordedAt) &&
    (value.revealedAt === undefined || isDate(value.revealedAt));
}

function isPlayer(p: unknown): boolean {
  return isRecord(p) && isHex(p.playerId, 32) && typeof p.name === 'string' && isTeam(p.team) &&
    typeof p.ready === 'boolean' && (p.route === undefined || isRoute(p.route)) &&
    (p.position === undefined || isPosition(p.position));
}

function isPlayground(p: unknown): p is Playground {
  const c = isRecord(p) ? p.center : undefined;
  const b = isRecord(p) ? p.bounds : undefined;
  return isRecord(p) && p.rows === 6 && p.columns === 6 && isValidCellSize(p.cellSizeMeters) &&
    isRecord(c) && finite(c.latitude) && Math.abs(c.latitude) <= 80 &&
    finite(c.longitude) && Math.abs(c.longitude) <= 180 &&
    isRecord(b) && finite(b.north) && finite(b.south) && finite(b.east) && finite(b.west) &&
    b.north <= 90 && b.south >= -90 && b.east <= 180 && b.west >= -180 &&
    b.south < c.latitude && b.north > c.latitude && b.west < c.longitude && b.east > c.longitude;
}

// Generated TypeScript types are not runtime validation of remote JSON.
function assertGameState(body: unknown): asserts body is GameStateResponse {
  if (!isRecord(body) || !isHex(body.sessionId, 32) || !['lobby', 'countdown', 'running'].includes(body.status as string) ||
      !isDate(body.serverTime) || (body.startsAt !== undefined && !isDate(body.startsAt)) ||
      (body.nextRevealAt !== undefined && !isDate(body.nextRevealAt)) ||
      !isPlayground(body.playground) || !isHex(body.playerId, 32) || !isTeam(body.team) || !isJoinCode(body.joinCode) ||
      (body.route !== undefined && (body.team !== 'runner' || !isRoute(body.route))) ||
      !Array.isArray(body.players) || !body.players.every(isPlayer)) {
    throw new Error('Unexpected game state. Check the API URL.');
  }
}

type Result = { data?: unknown; error?: unknown; response?: Response };

// Runs a generated SDK call with a five-second timeout and turns error
// responses into ApiError, keeping the backend's readable message.
async function request<R extends Result>(call: (signal: AbortSignal) => Promise<R>): Promise<NonNullable<R['data']>> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 5000);
  try {
    const result = await call(controller.signal);
    // Network failures and aborts are returned, not thrown, by the generated client.
    if (result.error instanceof Error) {
      throw result.error;
    }
    const status = result.response?.status ?? 0;
    if (result.error !== undefined || status < 200 || status > 299) {
      const message = isRecord(result.error) && typeof result.error.error === 'string'
        ? result.error.error
        : `Request failed with status ${status}.`;
      throw new ApiError(message, status);
    }
    return result.data as NonNullable<R['data']>;
  } catch (error) {
    if (controller.signal.aborted) {
      throw new Error('Connection timed out. Check the API URL and your network.');
    }
    if (error instanceof Error) {
      throw error;
    }
    throw new Error('Could not reach the backend. Check the API URL and your network.');
  } finally {
    clearTimeout(timeout);
  }
}
