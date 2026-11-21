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
};

/** The map's camera: refits it to the grid, keeps the grid centered on pan/tap and measures where the center is on screen. */
export function useMapRegion({ playground, editable, onSelect, onPick }: Options) {
  // Where the map's center (and so the grid) is on screen. Native maps shift it from the middle of the
  // view, e.g. by safe-area margins under the notch, so the crosshair is placed by asking the map.
  const [anchor, setAnchor] = useState<{ x: number; y: number } | null>(null);
  const map = useRef<MapView>(null);
  const [ready, setReady] = useState(false);
  const bounds = playground?.bounds;
  const playgroundRef = useRef(playground);
  useEffect(() => { playgroundRef.current = playground; });
  // Apple Maps (iOS) does not report isGesture reliably, so moves we cause ourselves (initial fit, refit, tap)
  // are muted for a moment instead.
  const muted = useRef(false);
  const mute = useRef((ms: number) => {
    muted.current = true;
    setTimeout(() => { muted.current = false; }, ms);
  });
  // Refit when the grid size or rotation changes, not while it is dragged: its span stays the same.
  const span = bounds ? bounds.north - bounds.south : undefined;
  const rotation = playground?.rotationDegrees;
  useEffect(() => {
    if (ready && playgroundRef.current) {
      mute.current(1200);
      map.current?.fitToCoordinates(playgroundCorners(playgroundRef.current), {
        edgePadding: { top: 24, bottom: 24, left: 24, right: 24 }, animated: true,
      });
    }
  }, [ready, span, rotation]);
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
    if (!editable || muted.current) return;
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
  return { map, anchor, select, onPress, onLayout, onMapReady, onRegionChangeComplete };
}
