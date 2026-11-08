import * as Location from 'expo-location';
import { useEffect, useRef, useState } from 'react';

import { isBackgroundSharingAvailable, startBackgroundSharing, stopBackgroundSharing } from './background-location';

const SEND_INTERVAL_MS = 5000;
// Browsers may not report accuracy; treat it as poor rather than precise.
const UNKNOWN_ACCURACY_METERS = 1000;

export type Fix = { latitude: number; longitude: number; accuracyMeters: number };

/**
 * While `active` and permitted, shares the position:
 * - With `background` (game running), background permission, and a build that
 *   supports it, a background task keeps reporting while the screen is off.
 * - Otherwise it watches in the foreground only and sends the latest fix
 *   immediately and then every five seconds.
 * Everything stops when inactive or unmounted. Nothing is stored on the device.
 */
export function useLocationSharing(active: boolean, background: boolean, send: (fix: Fix) => Promise<void>) {
  const [permission, requestPermission] = Location.useForegroundPermissions();
  const [backgroundPermission, requestBackgroundPermission] = Location.useBackgroundPermissions();
  const [backgroundAvailable, setBackgroundAvailable] = useState(false);
  const [error, setError] = useState('');
  const sendRef = useRef(send);
  const granted = permission?.granted ?? false;
  const inBackground = active && granted && background && backgroundAvailable && (backgroundPermission?.granted ?? false);

  useEffect(() => {
    sendRef.current = send;
  });

  useEffect(() => {
    let current = true;
    void isBackgroundSharingAvailable().then((available) => { if (current) setBackgroundAvailable(available); });
    return () => { current = false; };
  }, []);

  useEffect(() => {
    if (!inBackground) return;
    startBackgroundSharing().catch((err: unknown) => {
      setError(err instanceof Error ? err.message : 'Background location could not start.');
    });
    return () => { void stopBackgroundSharing(); };
  }, [inBackground]);

  useEffect(() => {
    if (!active || !granted || inBackground) return;
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
  }, [active, granted, inBackground]);

  return {
    permission, requestPermission, error,
    background: { available: backgroundAvailable, permission: backgroundPermission, request: requestBackgroundPermission, active: inBackground },
  };
}
