import { useEffect, useRef, useState } from 'react';
import type MapView from 'react-native-maps';
import type { LatLng, MapPressEvent, Region } from 'react-native-maps';

import type { Coordinates, Playground } from './api/types';
import { playgroundCorners } from './playground';

type Options = {
  playground: Playground | null;
  editable: boolean;
  onSelect: (center: Coordinates) => void;
  onPick?: (point: Coordinates) => void;
  /** Where the map's top should point after each refit, in degrees clockwise from north; unset keeps north-up. */
  heading?: number;
};

// How long after a refit the map is turned back: Android's refit always turns the map north-up first.
const TURN_BACK_AFTER_MS = 900;

/** The map's camera: refits it to the grid, keeps the grid centered on pan/tap and measures where the center is on screen. */
export function useMapRegion({ playground, editable, onSelect, onPick, heading }: Options) {
  // Where the map's center (and so the grid) is on screen. Native maps shift it from the middle of the
  // view, e.g. by safe-area margins under the notch, so the crosshair is placed by asking the map.
  const [anchor, setAnchor] = useState<{ x: number; y: number } | null>(null);
  const map = useRef<MapView>(null);
  const [ready, setReady] = useState(false);
  const bounds = playground?.bounds;
  const playgroundRef = useRef(playground);
  const headingRef = useRef(heading);
  useEffect(() => {
    playgroundRef.current = playground;
    headingRef.current = heading;
  });
  // Apple Maps (iOS) does not report isGesture reliably, so moves we cause ourselves (initial fit, refit, tap,
  // turning back) are muted for a moment instead. Overlapping mutes last until the latest one ends.
  const mutedUntil = useRef(0);
  const mute = useRef((ms: number) => { mutedUntil.current = Math.max(mutedUntil.current, Date.now() + ms); });
  const isMuted = () => Date.now() < mutedUntil.current;
  // Refit when the grid size changes, not while it is dragged or turned: its span stays the same.
  const span = bounds ? bounds.north - bounds.south : undefined;
  useEffect(() => {
    if (!ready || !playgroundRef.current) return;
    mute.current(TURN_BACK_AFTER_MS + 900);
    map.current?.fitToCoordinates(playgroundCorners(playgroundRef.current), {
      edgePadding: { top: 24, bottom: 24, left: 24, right: 24 }, animated: true,
    });
    if (!headingRef.current) return;
    const timer = setTimeout(() => {
      map.current?.animateCamera({ heading: headingRef.current ?? 0 }, { duration: 300 });
    }, TURN_BACK_AFTER_MS);
    return () => clearTimeout(timer);
  }, [ready, span]);
  const measureAnchor = async () => {
    try {
      const camera = await map.current?.getCamera();
      if (camera) setAnchor(await map.current!.pointForCoordinate(camera.center));
    } catch {
      // keep the middle of the view
    }
  };
  // Panning moves the grid: it stays centered on the map. While picking, the grid stays and the center is the point.
  const onRegionChangeComplete = (region: Region) => {
    if (onPick) onPick({ latitude: region.latitude, longitude: region.longitude });
    if (!editable || isMuted()) return;
    onSelect({ latitude: region.latitude, longitude: region.longitude });
  };
  // A tap places the grid there and scrolls the map to it, so a following pan does not jump.
  const select = (coordinate: LatLng) => {
    if (!editable) return;
    onSelect(coordinate);
    mute.current(800);
    map.current?.animateCamera({ center: coordinate }, { duration: 300 });
  };
  const onPress = (event: MapPressEvent) => select(event.nativeEvent.coordinate);
  const onLayout = () => setTimeout(() => void measureAnchor(), 300);
  const onMapReady = () => { setReady(true); void measureAnchor(); };
  return { map, anchor, select, onPress, onLayout, onMapReady, onRegionChangeComplete, isMuted };
}
