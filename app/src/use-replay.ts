import { useEffect, useMemo, useState } from 'react';

import type { GameResultResponse } from './api/types';
import type { MapMarker } from './components/playground-map';
import { routeHighlights, routeMarkers } from './lobby';
import { positionAt, REPLAY_SPEEDS, replayLegend, replayTracks, tailAt } from './replay';

/** How often the playing replay moves on, in real milliseconds. */
const TICK_MS = 100;

/** The replay clock (game milliseconds since the start) and what the map shows at that time. */
export function useReplay(result: GameResultResponse) {
  const duration = Date.parse(result.endedAt) - Date.parse(result.startsAt);
  const tracks = useMemo(() => replayTracks(result), [result]);
  const [at, setAt] = useState(0);
  const [playing, setPlaying] = useState(false);
  const [speed, setSpeed] = useState<number>(REPLAY_SPEEDS[1]);
  // Stops by itself at the end; pressing play there starts over.
  const running = playing && at < duration;

  useEffect(() => {
    if (!running) return;
    const timer = setInterval(() => setAt((previous) => Math.min(previous + TICK_MS * speed, duration)), TICK_MS);
    return () => clearInterval(timer);
  }, [running, speed, duration]);

  const toggle = () => {
    if (running) {
      setPlaying(false);
      return;
    }
    if (at >= duration) setAt(0);
    setPlaying(true);
  };
  const seek = (time: number) => setAt(Math.min(Math.max(time, 0), duration));

  const markers: MapMarker[] = [
    ...routeMarkers(result.route),
    ...tracks.flatMap((track) => {
      const coordinate = positionAt(track.points, at);
      return coordinate ? [{ id: track.playerId, coordinate, title: track.name, description: '', color: track.color, label: [...track.name][0].toUpperCase(), dot: true }] : [];
    }),
  ];
  const lines = tracks
    .map((track) => ({ id: track.playerId, coordinates: tailAt(track.points, at), color: track.color }))
    .filter((line) => line.coordinates.length >= 2);
  const highlights = useMemo(() => routeHighlights(result.playground, result.route), [result]);
  const legend = useMemo(() => replayLegend(result), [result]);

  return { at, duration, running, speed, setSpeed, toggle, seek, markers, lines, highlights, legend };
}
