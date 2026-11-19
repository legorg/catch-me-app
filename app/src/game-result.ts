import type { GameResultResponse, PositionResponse, ResultPlayerResponse } from './api/types';
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

function isPosition(value: unknown): value is PositionResponse {
  return isObject(value)
    && typeof value.latitude === 'number' && Math.abs(value.latitude) <= 90
    && typeof value.longitude === 'number' && Math.abs(value.longitude) <= 180
    && typeof value.accuracyMeters === 'number' && value.accuracyMeters >= 0 && Number.isFinite(value.accuracyMeters)
    && isTimestamp(value.recordedAt);
}

/** Runners and hunters always have a track (possibly empty); spectators never do. */
function isTrack(value: unknown): value is PositionResponse[] {
  return Array.isArray(value) && value.every(isPosition);
}

function isResultPlayer(value: unknown): value is ResultPlayerResponse {
  if (!isObject(value) || !isHexId(value.playerId) || typeof value.name !== 'string' || !value.name.trim()) return false;
  if (value.team === 'hunter') return value.runner === undefined && (value.hunterTeam === undefined || typeof value.hunterTeam === 'string') && isTrack(value.track);
  if (value.team === 'spectator') return value.runner === undefined && value.hunterTeam === undefined && value.track === undefined;
  return value.team === 'runner' && isObject(value.runner)
    && (value.runner.status === 'caught' || value.runner.status === 'finished') && isTimestamp(value.runner.endedAt)
    && isTrack(value.track);
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
