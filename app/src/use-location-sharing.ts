import * as Location from 'expo-location';
import { useEffect, useRef, useState } from 'react';
import { AppState } from 'react-native';

import { isBackgroundSharingAvailable, startBackgroundSharing, stopBackgroundSharing } from './background-location';
import { translate } from './i18n';

const SEND_INTERVAL_MS = 5000;
// A fix without accuracy is treated as poor rather than precise.
const UNKNOWN_ACCURACY_METERS = 1000;

export type LocationSharing = ReturnType<typeof useLocationSharing>;

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
  const [backgroundPermission, requestBackgroundPermission, getBackgroundPermission] = Location.useBackgroundPermissions();
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

  // "Always" is often granted in the system settings (Android 11+ opens them to ask): read it again on return.
  useEffect(() => {
    const subscription = AppState.addEventListener('change', (state) => {
      if (state === 'active') void getBackgroundPermission();
    });
    return () => subscription.remove();
  }, [getBackgroundPermission]);

  useEffect(() => {
    if (!inBackground) return;
    startBackgroundSharing().catch((err: unknown) => {
      setError(err instanceof Error ? err.message : translate('location.backgroundFailed'));
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
      (err: unknown) => setError(err instanceof Error ? err.message : translate('location.unavailable')),
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
