import type { GameResultResponse, ResultPlayerResponse } from './api/types';
import { getLanguage } from './i18n';
import { isHexId, isObject, isTimestamp } from './json-guards';
import { isTimelineEvent } from './timeline';

export function normalizeGameId(value: string): string {
  return value.trim().toLowerCase();
}

export function isValidGameId(value: string): boolean {
  return isHexId(value);
}

export function gameResultRoute(sessionId: string) {
  return { pathname: '/results/[sessionId]', params: { sessionId } } as const;
}

function isResultPlayer(value: unknown): value is ResultPlayerResponse {
  if (!isObject(value) || !isHexId(value.playerId) || typeof value.name !== 'string' || !value.name.trim()) return false;
  if (value.team === 'hunter') return value.runner === undefined && (value.hunterTeam === undefined || typeof value.hunterTeam === 'string');
  if (value.team === 'spectator') return value.runner === undefined && value.hunterTeam === undefined;
  return value.team === 'runner' && isObject(value.runner)
    && (value.runner.status === 'caught' || value.runner.status === 'finished') && isTimestamp(value.runner.endedAt);
}

/** Generated TypeScript types alone do not validate the JSON received at runtime. */
export function isGameResultResponse(value: unknown): value is GameResultResponse {
  return isObject(value) && isHexId(value.sessionId)
    && isTimestamp(value.startsAt) && isTimestamp(value.endedAt)
    && typeof value.durationSeconds === 'number' && Number.isSafeInteger(value.durationSeconds) && value.durationSeconds >= 0
    && Array.isArray(value.players) && value.players.every(isResultPlayer)
    && Array.isArray(value.events) && value.events.every(isTimelineEvent);
}

export function formatResultTime(value: string): string {
  return new Date(value).toLocaleString(getLanguage(), { dateStyle: 'medium', timeStyle: 'medium' });
}
