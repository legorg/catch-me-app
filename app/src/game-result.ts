import { getLanguage } from './i18n';
import { isHexId } from './json-guards';

export function normalizeGameId(value: string): string {
  return value.trim().toLowerCase();
}

export function isValidGameId(value: string): boolean {
  return isHexId(value);
}

export function gameResultRoute(sessionId: string) {
  return { pathname: '/results/[sessionId]', params: { sessionId } } as const;
}

export function formatResultTime(value: string): string {
  return new Date(value).toLocaleString(getLanguage(), { dateStyle: 'medium', timeStyle: 'medium' });
}
