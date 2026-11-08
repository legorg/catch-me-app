import * as Location from 'expo-location';
import { useEffect, useRef, useState } from 'react';

const SEND_INTERVAL_MS = 5000;
// Browsers may not report accuracy; treat it as poor rather than precise.
const UNKNOWN_ACCURACY_METERS = 1000;

export type Fix = { latitude: number; longitude: number; accuracyMeters: number };

/**
 * While `active` and permitted, watches the position in the foreground only and
 * sends the latest fix immediately and then every five seconds, so the server
 * keeps a fresh position even when the player stands still. Stops when
 * inactive, unfocused, or unmounted. Nothing is stored on the device.
 */
export function useLocationSharing(active: boolean, send: (fix: Fix) => Promise<void>) {
  const [permission, requestPermission] = Location.useForegroundPermissions();
  const [error, setError] = useState('');
  const sendRef = useRef(send);
  const granted = permission?.granted ?? false;

  useEffect(() => {
    sendRef.current = send;
  });

  useEffect(() => {
    if (!active || !granted) return;
    let latest: Fix | null = null;
    let subscription: Location.LocationSubscription | null = null;
    let stopped = false;
    const deliver = () => {
      if (latest) void sendRef.current(latest);
    };
    Location.watchPositionAsync(
      { accuracy: Location.Accuracy.High, timeInterval: 2000, distanceInterval: 0 },
      ({ coords }) => {
        const first = latest === null;
        latest = { latitude: coords.latitude, longitude: coords.longitude, accuracyMeters: coords.accuracy ?? UNKNOWN_ACCURACY_METERS };
        setError('');
        if (first) deliver();
      },
      (reason) => setError(reason),
    ).then(
      (s) => { if (stopped) s.remove(); else subscription = s; },
      (err: unknown) => setError(err instanceof Error ? err.message : 'Location is unavailable.'),
    );
    const timer = setInterval(deliver, SEND_INTERVAL_MS);
    return () => {
      stopped = true;
      subscription?.remove();
      clearInterval(timer);
    };
  }, [active, granted]);

  return { permission, requestPermission, error };
}
