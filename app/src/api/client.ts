import { createGame as postGame, getGame, getHealth } from './generated/sdk.gen';
import type { Coordinates, GameResponse, HealthResponse } from './generated/types.gen';
import { DEFAULT_CELL_SIZE_METERS, isValidCellSize } from '../playground';

export type { Coordinates, GameResponse, HealthResponse } from './generated/types.gen';

export const API_URL = (
  process.env.EXPO_PUBLIC_API_URL || 'http://localhost:8080'
).trim().replace(/\/+$/, '');

export async function checkBackend(): Promise<HealthResponse> {
  return withTimeout(async (signal) => {
    const { data } = await getHealth({ baseUrl: API_URL, signal, throwOnError: true });
    const body: unknown = data;
    if (
      typeof body !== 'object' ||
      body === null ||
      !('status' in body) ||
      body.status !== 'ok' ||
      !('service' in body) ||
      body.service !== 'catch-me-backend'
    ) {
      throw new Error('Unexpected backend response. Check the API URL.');
    }
    return data;
  });
}

export async function createGame(center: Coordinates, cellSizeMeters = DEFAULT_CELL_SIZE_METERS): Promise<GameResponse> {
  if (!isValidCellSize(cellSizeMeters)) {
    throw new Error('Cell size must be 0.1–5 km in steps of 0.1 km.');
  }
  return withTimeout(async (signal) => {
    const { data } = await postGame({
      baseUrl: API_URL,
      body: { ...center, cellSizeMeters },
      signal,
      throwOnError: true,
    });
    assertGameResponse(data);
    return data;
  });
}

export async function loadGame(sessionId: string): Promise<GameResponse> {
  if (!/^[a-f0-9]{32}$/.test(sessionId)) {
    throw new Error('Enter a session ID containing 32 lowercase hexadecimal characters.');
  }
  return withTimeout(async (signal) => {
    const { data } = await getGame({ baseUrl: API_URL, path: { sessionId }, signal, throwOnError: true });
    assertGameResponse(data);
    if (data.sessionId !== sessionId) {
      throw new Error('Backend returned a different game. Check the API URL.');
    }
    return data;
  });
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function finite(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value);
}

// Generated TypeScript types are not runtime validation of remote JSON.
function assertGameResponse(body: unknown): asserts body is GameResponse {
  const p = isRecord(body) ? body.playground : undefined;
  const c = isRecord(p) ? p.center : undefined;
  const b = isRecord(p) ? p.bounds : undefined;
  if (!isRecord(body) || typeof body.sessionId !== 'string' || !/^[a-f0-9]{32}$/.test(body.sessionId) ||
      typeof body.createdAt !== 'string' || !Number.isFinite(Date.parse(body.createdAt)) ||
      !isRecord(p) || p.rows !== 6 || p.columns !== 6 || !isValidCellSize(p.cellSizeMeters) ||
      !isRecord(c) || !finite(c.latitude) || Math.abs(c.latitude) > 80 ||
      !finite(c.longitude) || Math.abs(c.longitude) > 180 ||
      !isRecord(b) || !finite(b.north) || !finite(b.south) || !finite(b.east) || !finite(b.west) ||
      b.north > 90 || b.south < -90 || b.east > 180 || b.west < -180 ||
      b.south >= c.latitude || b.north <= c.latitude || b.west >= c.longitude || b.east <= c.longitude) {
    throw new Error('Unexpected game response. Check the API URL.');
  }
}

async function withTimeout<T>(request: (signal: AbortSignal) => Promise<T>): Promise<T> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 5000);

  try {
    return await request(controller.signal);
  } catch (error) {
    if (controller.signal.aborted) {
      throw new Error('Connection timed out. Check the API URL and your network.');
    }
    if (error instanceof Error) {
      throw error;
    }
    if (typeof error === 'object' && error !== null && 'error' in error && typeof error.error === 'string') {
      throw new Error(error.error);
    }
    throw new Error('Could not reach the backend. Check the API URL and your network.');
  } finally {
    clearTimeout(timeout);
  }
}
