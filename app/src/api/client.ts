import {
  activateJoker as postJoker,
  addSketch as postSketch,
  clearSketches as deleteSketches,
  createGame as postGame,
  getGameState as fetchGameState,
  getHealth,
  joinGame as postPlayer,
  leaveGame as deletePlayer,
  setHunterTeam as putHunterTeam,
  setReady as putReady,
  updatePosition as putPosition,
} from './generated/sdk.gen';
import { client } from './generated/client.gen';
import type { Coordinates, CreateGameResponse, GameRoute, GameStateResponse, HealthResponse, JoinCodes, JoinGameResponse, Playground } from './types';
import { translate } from '../i18n';
import { DEFAULT_CELL_SIZE_METERS, isValidCellSize } from '../playground';

export type {
  Coordinates, CreateGameResponse, GameRoute, GameStateResponse, HealthResponse, JoinCodes, JoinGameResponse, JokerPositionResponse,
  JokerResponse, Landmark, PlayerResponse, Playground, PositionResponse, SketchResponse,
} from './types';

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
  const data = await request((signal) => getHealth({ baseUrl: API_URL, signal }));
  if (!isRecord(data) || data.status !== 'ok' || data.service !== 'catch-me-backend') {
    throw new Error(translate('client.unexpectedHealth'));
  }
  return data;
}

export async function createGame(center: { latitude: number; longitude: number }, cellSizeMeters = DEFAULT_CELL_SIZE_METERS): Promise<CreateGameResponse> {
  if (!isValidCellSize(cellSizeMeters)) {
    throw new Error(translate('playground.cellSizeInvalid'));
  }
  const data = await request((signal) => postGame({ baseUrl: API_URL, body: { ...center, cellSizeMeters }, signal }));
  const game = isRecord(data) ? data.game : undefined;
  const codes = isRecord(data) ? data.joinCodes : undefined;
  if (!isRecord(game) || !isHex(game.sessionId, 32) || !isDate(game.createdAt) || !isPlayground(game.playground) ||
      !isGameRoute(game.route) || !isJoinCodes(codes)) {
    throw new Error(translate('client.unexpectedGame'));
  }
  return data as CreateGameResponse;
}

export async function joinGame(code: string, name: string): Promise<JoinGameResponse> {
  const normalized = normalizeJoinCode(code);
  if (!JOIN_CODE_PATTERN.test(normalized)) {
    throw new Error(translate('client.joinCode'));
  }
  if (!isValidPlayerName(name)) {
    throw new Error(translate('client.name', { max: MAX_NAME_LENGTH }));
  }
  const data = await request((signal) => postPlayer({ baseUrl: API_URL, body: { code: normalized, name: name.trim() }, signal }));
  if (!isRecord(data) || !isHex(data.playerToken, 64)) {
    throw new Error(translate('client.unexpectedJoin'));
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

export async function setHunterTeam(token: string, name: string): Promise<GameStateResponse> {
  if (!isValidPlayerName(name)) {
    throw new Error(translate('client.teamName', { max: MAX_NAME_LENGTH }));
  }
  const data = await request((signal) => putHunterTeam({ baseUrl: API_URL, auth: bearer(token), body: { name: name.trim() }, signal }));
  assertGameState(data);
  return data;
}

export type JokerKind = 'showRunners';

/** Spends a joker of the player's hunter team (once per game, while it runs). */
export async function activateJoker(token: string, kind: JokerKind): Promise<GameStateResponse> {
  const data = await request((signal) => postJoker({ baseUrl: API_URL, auth: bearer(token), body: { kind }, signal }));
  assertGameState(data);
  return data;
}

export const MIN_SKETCH_POINTS = 2;
export const MAX_SKETCH_POINTS = 500;

/** Draws a line on the map of the player's hunter team (while the game runs). */
export async function addSketch(token: string, points: Coordinates[]): Promise<GameStateResponse> {
  if (points.length < MIN_SKETCH_POINTS || points.length > MAX_SKETCH_POINTS) {
    throw new Error(translate('client.sketchPoints', { min: MIN_SKETCH_POINTS, max: MAX_SKETCH_POINTS }));
  }
  const body = { points: points.map(({ latitude, longitude }) => ({ latitude, longitude })) };
  const data = await request((signal) => postSketch({ baseUrl: API_URL, auth: bearer(token), body, signal }));
  assertGameState(data);
  return data;
}

/** Removes all lines of the player's hunter team, whoever drew them. */
export async function clearSketches(token: string): Promise<GameStateResponse> {
  const data = await request((signal) => deleteSketches({ baseUrl: API_URL, auth: bearer(token), signal }));
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

function isJoinCodes(value: unknown): value is JoinCodes {
  return isRecord(value) && isJoinCode(value.runner) && isJoinCode(value.hunter) && value.runner !== value.hunter;
}

function isTeam(value: unknown): value is 'runner' | 'hunter' {
  return value === 'runner' || value === 'hunter';
}

function isLandmark(value: unknown): boolean {
  const c = isRecord(value) ? value.coordinates : undefined;
  return isRecord(value) && typeof value.title === 'string' &&
    isRecord(c) && finite(c.latitude) && Math.abs(c.latitude) <= 80 && finite(c.longitude) && Math.abs(c.longitude) <= 180;
}

function isGameRoute(value: unknown): value is GameRoute {
  return isRecord(value) && isLandmark(value.start) && isLandmark(value.stop) &&
    finite(value.distanceMeters) && value.distanceMeters >= 0;
}

function isPosition(value: unknown): boolean {
  return isRecord(value) && finite(value.latitude) && Math.abs(value.latitude) <= 90 &&
    finite(value.longitude) && Math.abs(value.longitude) <= 180 &&
    finite(value.accuracyMeters) && value.accuracyMeters >= 0 && isDate(value.recordedAt);
}

function isReveal(value: unknown): boolean {
  return isPosition(value) && isDate((value as Record<string, unknown>).revealedAt);
}

function isJoker(j: unknown): boolean {
  return isRecord(j) && j.kind === 'showRunners' && typeof j.available === 'boolean' &&
    (j.usedAt === undefined || isDate(j.usedAt)) &&
    (j.positions === undefined || (Array.isArray(j.positions) && j.positions.every((p) => isRecord(p) && isHex(p.playerId, 32) && isPosition(p))));
}

function isSketch(s: unknown): boolean {
  return isRecord(s) && isHex(s.sketchId, 32) && isDate(s.createdAt) && Array.isArray(s.points) &&
    s.points.length >= MIN_SKETCH_POINTS && s.points.length <= MAX_SKETCH_POINTS &&
    s.points.every((p) => isRecord(p) && finite(p.latitude) && Math.abs(p.latitude) <= 80 && finite(p.longitude) && Math.abs(p.longitude) <= 180);
}

function isPlayer(p: unknown): boolean {
  return isRecord(p) && isHex(p.playerId, 32) && typeof p.name === 'string' && isTeam(p.team) &&
    (p.hunterTeam === undefined || (p.team === 'hunter' && typeof p.hunterTeam === 'string')) &&
    typeof p.ready === 'boolean' &&
    (p.position === undefined || isPosition(p.position)) &&
    (p.reveals === undefined || (Array.isArray(p.reveals) && p.reveals.every(isReveal)));
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
      (body.startPositionCheckDisabled !== undefined && typeof body.startPositionCheckDisabled !== 'boolean') ||
      (body.nextRevealAt !== undefined && !isDate(body.nextRevealAt)) ||
      !isPlayground(body.playground) || !isHex(body.playerId, 32) || !isTeam(body.team) || !isJoinCodes(body.joinCodes) ||
      // Only runners get the route; hunters must not.
      (body.team === 'runner' ? !isGameRoute(body.route) : body.route !== undefined) ||
      !Array.isArray(body.players) || !body.players.every(isPlayer) ||
      !Array.isArray(body.jokers) || !body.jokers.every(isJoker) ||
      // Only hunters get their team's sketches; runners never.
      !Array.isArray(body.sketches) || !body.sketches.every(isSketch) || (body.team === 'runner' && body.sketches.length > 0)) {
    throw new Error(translate('client.unexpectedState'));
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
        : translate('client.status', { status });
      throw new ApiError(message, status);
    }
    return result.data as NonNullable<R['data']>;
  } catch (error) {
    console.log(error)
    if (controller.signal.aborted) {
      throw new Error(translate('client.timeout'));
    }
    if (error instanceof Error) {
      throw error;
    }
    throw new Error(translate('client.unreachable'));
  } finally {
    clearTimeout(timeout);
  }
}
