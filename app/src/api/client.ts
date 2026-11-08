import { getDemoGreeting, getHealth } from './generated/sdk.gen';
import type { DemoResponse, HealthResponse } from './generated/types.gen';

export type { DemoResponse, HealthResponse } from './generated/types.gen';

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

export async function loadDemoGreeting(name = 'Runner'): Promise<DemoResponse> {
  return withTimeout(async (signal) => {
    const { data } = await getDemoGreeting({
      baseUrl: API_URL,
      query: { name },
      signal,
      throwOnError: true,
    });
    // Generated TypeScript types are not runtime validation of remote JSON.
    const body: unknown = data;
    if (
      typeof body !== 'object' ||
      body === null ||
      !('message' in body) ||
      typeof body.message !== 'string' ||
      !('playerName' in body) ||
      typeof body.playerName !== 'string' ||
      body.playerName.length === 0 ||
      Array.from(body.playerName).length > 60 ||
      !('service' in body) ||
      body.service !== 'catch-me-backend'
    ) {
      throw new Error('Unexpected demo response. Check the API URL.');
    }
    return data;
  });
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
