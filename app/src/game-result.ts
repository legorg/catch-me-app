import { getLanguage } from './i18n';

export function normalizeGameId(value: string): string {
  return value.trim().toLowerCase();
}

/** Game IDs: 32 lowercase hexadecimal characters. */
export function isValidGameId(value: string): boolean {
  return /^[a-f0-9]{32}$/.test(value);
}

export function gameResultRoute(sessionId: string) {
  return { pathname: '/results/[sessionId]', params: { sessionId } } as const;
}

export function formatResultTime(value: string): string {
  return new Date(value).toLocaleString(getLanguage(), { dateStyle: 'medium', timeStyle: 'medium' });
}
