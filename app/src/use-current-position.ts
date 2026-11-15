import * as Location from 'expo-location';
import { useEffect, useState } from 'react';

import type { Coordinates } from '@/api/client';

/** The device's position once permission is granted and a fix arrives; null until then or when unavailable. */
export function useCurrentPosition() {
  const [position, setPosition] = useState<Coordinates | null>(null);

  useEffect(() => {
    let active = true;
    void (async () => {
      try {
        const { granted } = await Location.requestForegroundPermissionsAsync();
        if (!granted) return;
        const { coords } = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
        if (active) setPosition({ latitude: coords.latitude, longitude: coords.longitude });
      } catch {
        // stay at the default spot; the map's locate button still works
      }
    })();
    return () => { active = false; };
  }, []);

  return position;
}
