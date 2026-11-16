import { useRef, useState, type RefObject } from 'react';
import type { GestureResponderEvent } from 'react-native';
import type MapView from 'react-native-maps';
import type { LatLng } from 'react-native-maps';

import type { Coordinates } from './api/types';

// A finger must move this far (in points) before the line gets a new point, which keeps lines small.
const MIN_STROKE_STEP = 6;

type Stroke = { pending: Promise<LatLng | null>[]; resolved: (LatLng | null)[]; last?: { x: number; y: number } };

/** Turns finger moves over the map into a line; `responder` goes onto the view that catches the finger. */
export function useMapDrawing(map: RefObject<MapView | null>, onDraw?: (points: Coordinates[]) => Promise<void>) {
  // Drawing: each finger position becomes a map coordinate (asynchronously, so they are collected by index).
  const [stroke, setStroke] = useState<Coordinates[]>([]);
  const currentStroke = useRef<Stroke | null>(null);
  const addPoint = (x: number, y: number) => {
    const current = currentStroke.current;
    if (!current || !map.current || (current.last && Math.hypot(x - current.last.x, y - current.last.y) < MIN_STROKE_STEP)) return;
    current.last = { x, y };
    const index = current.pending.length;
    const point = map.current.coordinateForPoint({ x, y }).catch(() => null);
    current.pending.push(point);
    void point.then((coordinate) => {
      current.resolved[index] = coordinate;
      if (currentStroke.current === current) setStroke(current.resolved.filter((c): c is LatLng => !!c));
    });
  };
  // Uses the onDraw of the moment the finger lifted, so switching drawing off right after still sends the line.
  const finishStroke = async () => {
    const send = onDraw;
    const current = currentStroke.current;
    currentStroke.current = null;
    if (!current) return;
    const points = (await Promise.all(current.pending)).filter((c): c is LatLng => !!c);
    try {
      if (points.length >= 2) await send?.(points.map(({ latitude, longitude }) => ({ latitude, longitude })));
    } finally {
      // A new line may have been started meanwhile; it shows its own preview.
      if (!currentStroke.current) setStroke([]);
    }
  };
  const startStroke = (x: number, y: number) => {
    currentStroke.current = { pending: [], resolved: [] };
    addPoint(x, y);
  };
  const responder = {
    onStartShouldSetResponder: () => true,
    onMoveShouldSetResponder: () => true,
    onResponderGrant: (event: GestureResponderEvent) => startStroke(event.nativeEvent.locationX, event.nativeEvent.locationY),
    onResponderMove: (event: GestureResponderEvent) => addPoint(event.nativeEvent.locationX, event.nativeEvent.locationY),
    onResponderRelease: () => void finishStroke(),
    onResponderTerminate: () => void finishStroke(),
  };
  return { drawing: !!onDraw, stroke, responder };
}
