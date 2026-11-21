import { isRunningInExpoGo } from 'expo';
import * as Location from 'expo-location';
import * as TaskManager from 'expo-task-manager';

import { ApiError, updatePosition } from './api/client';
import { translate } from './i18n';
import { playerSession } from './player-session';
import { cancelRevealWarning, notifyGameAlerts } from './game-notifications';

const TASK_NAME = 'catch-me-background-location';
const UNKNOWN_ACCURACY_METERS = 1000;

// Serialize native start/stop calls: cleanup must not finish before a pending
// start registers its task, otherwise a confirmed catch could leave it running.
let sharingOperation: Promise<void> = Promise.resolve();
let sharingRequested = false;

type TaskData = { locations?: Location.LocationObject[] };

// Must run at module load (imported by the root layout), so the task exists
// when the operating system delivers locations while the app is in the background.
TaskManager.defineTask<TaskData>(TASK_NAME, async ({ data, error }) => {
  const token = playerSession.get();
  // Ignore queued fixes immediately once sharing stops, even if the native
  // stop is still pending. On restart, wait for the game screen to opt in.
  if (!token || !sharingRequested) {
    await stopBackgroundSharing();
    return;
  }
  const latest = data?.locations?.at(-1);
  if (error || !latest) return;
  try {
    // A report response may carry a new zone alert, showRunners use or triggered trap, and the next reveal
    // time; notify and schedule them even with the screen off.
    await notifyGameAlerts(await updatePosition(token, {
      latitude: latest.coords.latitude,
      longitude: latest.coords.longitude,
      accuracyMeters: latest.coords.accuracy ?? UNKNOWN_ACCURACY_METERS,
    }));
  } catch (err) {
    if (err instanceof ApiError && (err.status === 401 || err.status === 409)) {
      await stopBackgroundSharing();
      await cancelRevealWarning();
    }
    // Position updates return 409 after a catch/finish or game end (hunters
    // too), including when an action response was lost or the app is backgrounded.
    // Other failures (e.g. no network) are retried with the next location.
  }
});

/** Background location needs a development or production build; Expo Go cannot run it. */
export async function isBackgroundSharingAvailable(): Promise<boolean> {
  // Expo Go reports background location as available on iOS, but cannot run it.
  if (isRunningInExpoGo()) return false;
  try {
    return await TaskManager.isAvailableAsync() && await Location.isBackgroundLocationAvailableAsync();
  } catch {
    return false;
  }
}

export function startBackgroundSharing(): Promise<void> {
  sharingRequested = true;
  sharingOperation = sharingOperation.catch(() => {}).then(async () => {
    if (!sharingRequested) return;
    if (await Location.hasStartedLocationUpdatesAsync(TASK_NAME)) return;
    await Location.startLocationUpdatesAsync(TASK_NAME, {
      accuracy: Location.Accuracy.High,
      timeInterval: 5000,
      distanceInterval: 5,
      pausesUpdatesAutomatically: false,
      activityType: Location.ActivityType.Fitness,
      showsBackgroundLocationIndicator: true,
      // Uses the language active when sharing starts.
      foregroundService: {
        notificationTitle: translate('location.notificationTitle'),
        notificationBody: translate('location.notificationBody'),
        killServiceOnDestroy: true,
      },
    });
  });
  return sharingOperation;
}

export function stopBackgroundSharing(): Promise<void> {
  sharingRequested = false;
  sharingOperation = sharingOperation.catch(() => {}).then(async () => {
    try {
      if (await Location.hasStartedLocationUpdatesAsync(TASK_NAME)) {
        await Location.stopLocationUpdatesAsync(TASK_NAME);
      }
    } catch {
      // Already stopped or unavailable.
    }
  });
  return sharingOperation;
}
