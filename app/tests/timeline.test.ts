import assert from 'node:assert/strict';
import { test } from 'node:test';

import type { TimelineEvent, TimelineResponse } from '../src/api/types';
import { setLanguage } from '../src/i18n';
import { newestFirst, shouldAcceptTimeline, timelineKey, timelineText } from '../src/timeline';

const ada = { playerId: 'a'.repeat(32), name: 'Ada' };
const events: TimelineEvent[] = [
  { kind: 'gameStarted', at: '2026-11-18T12:00:10Z' },
  { kind: 'hunterJokerUsed', at: '2026-11-18T12:07:00Z', hunterTeam: 'Wolves', joker: 'showRunners' },
  { kind: 'runnerJokerUsed', at: '2026-11-18T12:08:00Z', runner: ada, joker: 'zoneOnly' },
  { kind: 'trapTriggered', at: '2026-11-18T12:20:00Z', hunterTeam: 'Wolves' },
  { kind: 'zoneAlert', at: '2026-11-18T12:25:00Z', runner: ada, hunter: { playerId: 'c'.repeat(32), name: 'Eve' }, hunterTeam: 'Wolves' },
  { kind: 'runnerCaught', at: '2026-11-18T12:30:00Z', runner: ada },
  { kind: 'runnerFinished', at: '2026-11-18T13:00:10Z', runner: { playerId: 'b'.repeat(32), name: 'Cy' } },
  { kind: 'gameEnded', at: '2026-11-18T13:00:10Z' },
];

test('the ticker shows the newest event first without changing the server order', () => {
  const shown = newestFirst(events);
  assert.deepEqual(shown.map((event) => event.kind), ['gameEnded', 'runnerFinished', 'runnerCaught', 'zoneAlert', 'trapTriggered', 'runnerJokerUsed', 'hunterJokerUsed', 'gameStarted']);
  assert.equal(events[0].kind, 'gameStarted');
});

test('every event has a readable text in each language', () => {
  assert.deepEqual(events.map(timelineText), [
    'The game started',
    'Hunter team Wolves used “Show runners”',
    'Ada used “Show only the zone”',
    'A runner walked into hunter team Wolves\'s trap',
    'Ada was warned: Eve (Wolves) was in the same zone',
    'Ada was caught',
    'Cy reached the destination',
    'The game ended',
  ]);
  setLanguage('de');
  try {
    assert.equal(timelineText(events[5]), 'Ada wurde gefangen');
    assert.equal(timelineText(events[3]), 'Ein Läufer ist in die Falle von Jägerteam Wolves getappt');
    assert.equal(timelineText(events[4]), 'Ada wurde gewarnt: Eve (Wolves) war in derselben Zone');
    assert.ok(!timelineText(events[1]).includes('{'));
  } finally {
    setLanguage('en');
  }
});

test('events at the same instant have distinct keys', () => {
  const keys = events.map(timelineKey);
  assert.equal(new Set(keys).size, keys.length);
  const sameInstant: TimelineEvent = { kind: 'hunterJokerUsed', at: events[1].at, hunterTeam: 'Wolves', joker: 'compass' };
  assert.notEqual(timelineKey(sameInstant), timelineKey(events[1]));
});

test('an older response does not replace a newer timeline', () => {
  const newer: TimelineResponse = { serverTime: '2026-11-18T12:30:05Z', events };
  const older: TimelineResponse = { serverTime: '2026-11-18T12:30:00.5Z', events: events.slice(0, 3) };
  assert.equal(shouldAcceptTimeline(null, older), true);
  assert.equal(shouldAcceptTimeline(newer, older), false);
  assert.equal(shouldAcceptTimeline(older, newer), true);
});
