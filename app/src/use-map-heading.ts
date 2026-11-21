import { useRef, useState, type RefObject } from 'react';
import type MapView from 'react-native-maps';
import type { Region } from 'react-native-maps';

type Options = {
  rotatable: boolean;
  onRegionChangeComplete: (region: Region) => void;
  /** Whether the map is moving because the app moved it (refit, tap, turning back). */
  isMuted: () => boolean;
  /** Called when the user turns the map (two fingers or the north button), with its new heading. */
  onHeadingChange?: (heading: number) => void;
};

/**
 * Where the top of a rotatable map points (degrees clockwise from north) and turning it. Wraps the map's
 * onRegionChangeComplete, as the map also comes to rest after a two-finger turn.
 */
export function useMapHeading(map: RefObject<MapView | null>, { rotatable, onRegionChangeComplete, isMuted, onHeadingChange }: Options) {
  const [heading, setHeading] = useState(0);
  // The last heading seen, including those the app caused itself, so only the user's turns are reported.
  const seen = useRef(0);
  const update = async () => {
    if (!rotatable) return;
    try {
      const camera = await map.current?.getCamera();
      if (!camera) return;
      setHeading(camera.heading);
      const turnedByUser = !isMuted() && Math.round(camera.heading) !== Math.round(seen.current);
      seen.current = camera.heading;
      if (turnedByUser) onHeadingChange?.(camera.heading);
    } catch {
      // keep the last known heading
    }
  };
  const turnTo = (degrees: number) => {
    map.current?.animateCamera({ heading: degrees }, { duration: 300 });
    setHeading(degrees);
    seen.current = degrees;
    onHeadingChange?.(degrees);
  };
  const onRest = (region: Region) => {
    onRegionChangeComplete(region);
    void update();
  };
  // Within half a degree of a direction counts as pointing there.
  const pointsTo = (degrees: number) => Math.abs(((heading - degrees) % 360 + 540) % 360 - 180) < 0.5;
  return { heading, pointsTo, turnTo, onRegionChangeComplete: onRest };
}
