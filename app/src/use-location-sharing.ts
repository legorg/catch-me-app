import * as Location from 'expo-location';
import { useEffect, useState } from 'react';
import { AppState } from 'react-native';

import { isBackgroundSharingAvailable, startBackgroundSharing, stopBackgroundSharing } from './background-location';
import { translate } from './i18n';
import { dropLocation, offerLocation, type LocationReport } from './pending-location';

const SEND_INTERVAL_MS = 5000;
// A fix without accuracy is treated as poor rather than precise.
const UNKNOWN_ACCURACY_METERS = 1000;

export type LocationSharing = ReturnType<typeof useLocationSharing>;

/**
 * While `active` and permitted, shares the position:
 * - With `background` (game running), background permission, and a build that
 *   supports it, a background task keeps reporting while the screen is off.
 * - Otherwise it watches in the foreground only and leaves the latest location for
 *   the next state poll to report (see pending-location), at once and then every five seconds.
 * Everything stops when inactive or unmounted. Nothing is stored on the device.
 */
export function useLocationSharing(active: boolean, background: boolean) {
  const [permission, requestPermission] = Location.useForegroundPermissions();
  const [backgroundPermission, requestBackgroundPermission, getBackgroundPermission] = Location.useBackgroundPermissions();
  const [backgroundAvailable, setBackgroundAvailable] = useState(false);
  const [error, setError] = useState('');
  const granted = permission?.granted ?? false;
  const inBackground = active && granted && background && backgroundAvailable && (backgroundPermission?.granted ?? false);

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
    let latest: LocationReport | null = null;
    let subscription: Location.LocationSubscription | null = null;
    let stopped = false;
    const deliver = () => {
      if (!stopped && latest) offerLocation(latest);
    };
    Location.watchPositionAsync(
      { accuracy: Location.Accuracy.High, timeInterval: 2000, distanceInterval: 0 },
      ({ coords }) => {
        if (stopped) return;
        const first = latest === null;
        latest = { latitude: coords.latitude, longitude: coords.longitude, accuracyMeters: coords.accuracy ?? UNKNOWN_ACCURACY_METERS };
        setError('');
        if (first) deliver();
      },
      (reason) => { if (!stopped) setError(reason); },
    ).then(
      (s) => { if (stopped) s.remove(); else subscription = s; },
      (err: unknown) => { if (!stopped) setError(err instanceof Error ? err.message : translate('location.unavailable')); },
    );
    const timer = setInterval(deliver, SEND_INTERVAL_MS);
    return () => {
      stopped = true;
      dropLocation();
      subscription?.remove();
      clearInterval(timer);
    };
  }, [active, granted, inBackground]);

  return {
    permission, requestPermission, error,
    background: { available: backgroundAvailable, permission: backgroundPermission, request: requestBackgroundPermission, active: inBackground },
  };
}
