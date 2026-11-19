import type { Coordinates, GameResultResponse } from './api/types';
import { SPECTATOR_COLORS } from './lobby';

/** Game time shown behind each player as a tail. */
const TAIL_MS = 30 * 60_000;
/** Game seconds per real second. */
export const REPLAY_SPEEDS = [10, 60, 300] as const;
/** Hunter teams in order of their first member; used again from the start when there are more teams. */
const HUNTER_TEAM_COLORS = ['#eb5757', '#f2994a', '#9b51e0', '#00897b', '#6d4c41', '#e84393'];

type TimedPoint = { latitude: number; longitude: number; at: number };

export type ReplayTrack = { playerId: string; name: string; color: string; points: TimedPoint[] };

/** One legend row: the runners, or one hunter team (without a name for hunters who never chose one). */
export type ReplayLegendEntry = { key: string; team: 'runner' | 'hunter'; hunterTeam?: string; color: string };

/** Runners first, then every hunter team in join order of its first member, each with its color. */
export function replayLegend(result: GameResultResponse): ReplayLegendEntry[] {
  const legend: ReplayLegendEntry[] = [];
  if (result.players.some((p) => p.team === 'runner')) legend.push({ key: 'runner', team: 'runner', color: SPECTATOR_COLORS.runner });
  for (const p of result.players) {
    if (p.team !== 'hunter' || legend.some((entry) => entry.key === hunterKey(p.hunterTeam))) continue;
    const index = legend.filter((entry) => entry.team === 'hunter').length;
    legend.push({ key: hunterKey(p.hunterTeam), team: 'hunter', hunterTeam: p.hunterTeam, color: HUNTER_TEAM_COLORS[index % HUNTER_TEAM_COLORS.length] });
  }
  return legend;
}

function hunterKey(hunterTeam: string | undefined): string {
  return `hunter:${hunterTeam ?? ''}`;
}

/** Runners' and hunters' tracks with times in milliseconds since the start, in their legend color; spectators have none. */
export function replayTracks(result: GameResultResponse): ReplayTrack[] {
  const start = Date.parse(result.startsAt);
  const colors = new Map(replayLegend(result).map((entry) => [entry.key, entry.color]));
  return result.players.flatMap((p) => p.track && (p.team === 'runner' || p.team === 'hunter') ? [{
    playerId: p.playerId,
    name: p.name,
    color: colors.get(p.team === 'runner' ? 'runner' : hunterKey(p.hunterTeam)) ?? SPECTATOR_COLORS[p.team],
    points: p.track.map(({ latitude, longitude, recordedAt }) => ({ latitude, longitude, at: Date.parse(recordedAt) - start })),
  }] : []);
}

function between(a: TimedPoint, b: TimedPoint, at: number): Coordinates {
  const share = b.at === a.at ? 1 : (at - a.at) / (b.at - a.at);
  return { latitude: a.latitude + (b.latitude - a.latitude) * share, longitude: a.longitude + (b.longitude - a.longitude) * share };
}

/** Index of the last point at or before `at`, or -1. Points are oldest first. */
function lastIndexAt(points: TimedPoint[], at: number): number {
  let low = 0;
  let high = points.length - 1;
  while (low <= high) {
    const middle = (low + high) >> 1;
    if (points[middle].at <= at) low = middle + 1;
    else high = middle - 1;
  }
  return high;
}

/** Where the player was: hidden before the first point, linear between points, staying at the last one. */
export function positionAt(points: TimedPoint[], at: number): Coordinates | null {
  const index = lastIndexAt(points, at);
  if (index < 0) return null;
  const point = points[index];
  const next = points[index + 1];
  return next ? between(point, next, at) : { latitude: point.latitude, longitude: point.longitude };
}

/** The path covered during the `tailMs` before `at`, ending at the current position. */
export function tailAt(points: TimedPoint[], at: number, tailMs = TAIL_MS): Coordinates[] {
  const current = positionAt(points, at);
  if (!current) return [];
  const from = at - tailMs;
  const first = lastIndexAt(points, from);
  const start = first >= 0 ? positionAt(points, from) : null;
  const inside = points.slice(first + 1, lastIndexAt(points, at) + 1).map(({ latitude, longitude }) => ({ latitude, longitude }));
  return [...(start ? [start] : []), ...inside, current];
}
