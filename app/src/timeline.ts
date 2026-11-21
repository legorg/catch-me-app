import type { TimelineEvent, TimelineResponse } from './api/types';
import { translate } from './i18n';

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
    case 'trapTriggered':
      return `${event.kind}|${event.at}|${event.hunterTeam}`;
    case 'zoneAlert':
      return `${event.kind}|${event.at}|${event.runner.playerId}|${event.hunter.playerId}`;
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
    case 'trapTriggered':
      return translate('timeline.trapTriggered', { team: event.hunterTeam });
    case 'zoneAlert':
      return translate('timeline.zoneAlert', { name: event.runner.name, hunter: event.hunter.name, team: event.hunterTeam });
    case 'runnerCaught':
      return translate('timeline.runnerCaught', { name: event.runner.name });
    case 'runnerFinished':
      return translate('timeline.runnerFinished', { name: event.runner.name });
    case 'gameEnded':
      return translate('timeline.gameEnded');
  }
}
