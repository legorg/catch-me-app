import type { JokerKind, TimelineEvent, TimelinePlayer, TimelineResponse } from './api/types';
import { translate } from './i18n';
import { isHexId, isObject, isTimestamp } from './json-guards';

// Every joker kind; the type check reports a missing one.
const JOKER_KINDS = {
  showRunners: true, queryZone: true, zoneOnly: true, fakePosition: true, showHunters: true, compass: true,
} satisfies Record<JokerKind, true>;

function isJokerKind(value: unknown): value is JokerKind {
  return typeof value === 'string' && Object.hasOwn(JOKER_KINDS, value);
}

function isName(value: unknown): value is string {
  return typeof value === 'string' && !!value.trim();
}

function isTimelinePlayer(value: unknown): value is TimelinePlayer {
  return isObject(value) && isHexId(value.playerId) && isName(value.name);
}

/** Also used for the events of a game result. */
function isTimelineEvent(value: unknown): value is TimelineEvent {
  if (!isObject(value) || !isTimestamp(value.at)) return false;
  switch (value.kind) {
    case 'gameStarted':
    case 'gameEnded':
      return true;
    case 'hunterJokerUsed':
      return isName(value.hunterTeam) && isJokerKind(value.joker);
    case 'runnerJokerUsed':
      return isTimelinePlayer(value.runner) && isJokerKind(value.joker);
    case 'runnerCaught':
    case 'runnerFinished':
      return isTimelinePlayer(value.runner);
  }
  return false;
}

/** Generated TypeScript types alone do not validate the JSON received at runtime. */
export function isTimelineResponse(value: unknown): value is TimelineResponse {
  return isObject(value) && isTimestamp(value.serverTime) && Array.isArray(value.events) && value.events.every(isTimelineEvent);
}

/** A slower, older response must not replace a newer timeline. */
export function shouldAcceptTimeline(previous: TimelineResponse | null, incoming: TimelineResponse): boolean {
  return !previous || Date.parse(incoming.serverTime) >= Date.parse(previous.serverTime);
}

/** The server sends the oldest event first; the ticker shows the newest first. */
export function newestFirst(events: readonly TimelineEvent[]): TimelineEvent[] {
  return [...events].reverse();
}

/** Stable across polls, so list items keep their identity as new events arrive. */
export function timelineKey(event: TimelineEvent): string {
  switch (event.kind) {
    case 'hunterJokerUsed':
      return `${event.kind}|${event.at}|${event.hunterTeam}|${event.joker}`;
    case 'runnerJokerUsed':
      return `${event.kind}|${event.at}|${event.runner.playerId}|${event.joker}`;
    case 'runnerCaught':
    case 'runnerFinished':
      return `${event.kind}|${event.at}|${event.runner.playerId}`;
  }
  return `${event.kind}|${event.at}`;
}

export function timelineText(event: TimelineEvent): string {
  switch (event.kind) {
    case 'gameStarted':
      return translate('timeline.gameStarted');
    case 'hunterJokerUsed':
      return translate('timeline.hunterJokerUsed', { team: event.hunterTeam, joker: translate(`joker.${event.joker}.title`) });
    case 'runnerJokerUsed':
      return translate('timeline.runnerJokerUsed', { name: event.runner.name, joker: translate(`joker.${event.joker}.title`) });
    case 'runnerCaught':
      return translate('timeline.runnerCaught', { name: event.runner.name });
    case 'runnerFinished':
      return translate('timeline.runnerFinished', { name: event.runner.name });
    case 'gameEnded':
      return translate('timeline.gameEnded');
  }
}
