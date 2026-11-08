import * as Location from 'expo-location';
import * as TaskManager from 'expo-task-manager';
import { Platform } from 'react-native';

import { ApiError, updatePosition } from './api/client';
import { playerSession } from './player-session';

const TASK_NAME = 'catch-me-background-location';
const UNKNOWN_ACCURACY_METERS = 1000;

type TaskData = { locations?: Location.LocationObject[] };

// Must run at module load (imported by the root layout), so the task exists
// when the operating system delivers locations while the app is in the background.
if (Platform.OS !== 'web') {
  TaskManager.defineTask<TaskData>(TASK_NAME, async ({ data, error }) => {
    const token = playerSession.get();
    // No token means the player left, or the app was restarted and lost its
    // in-memory session: stop instead of tracking without purpose.
    if (!token) {
      await stopBackgroundSharing();
      return;
    }
    const latest = data?.locations?.at(-1);
    if (error || !latest) return;
    try {
      await updatePosition(token, {
        latitude: latest.coords.latitude,
        longitude: latest.coords.longitude,
        accuracyMeters: latest.coords.accuracy ?? UNKNOWN_ACCURACY_METERS,
      });
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) await stopBackgroundSharing();
      // Other failures (e.g. no network) are retried with the next location.
    }
  });
}

/** Background location needs a development or production build; Expo Go and the web cannot run it. */
export async function isBackgroundSharingAvailable(): Promise<boolean> {
  if (Platform.OS === 'web') return false;
  try {
    return await TaskManager.isAvailableAsync() && await Location.isBackgroundLocationAvailableAsync();
  } catch {
    return false;
  }
}

export async function startBackgroundSharing(): Promise<void> {
  if (await Location.hasStartedLocationUpdatesAsync(TASK_NAME)) return;
  await Location.startLocationUpdatesAsync(TASK_NAME, {
    accuracy: Location.Accuracy.High,
    timeInterval: 5000,
    distanceInterval: 5,
    pausesUpdatesAutomatically: false,
    activityType: Location.ActivityType.Fitness,
    showsBackgroundLocationIndicator: true,
    foregroundService: {
      notificationTitle: 'Catch me! is sharing your location',
      notificationBody: 'Your team sees your position while the game runs.',
      killServiceOnDestroy: true,
    },
  });
}

export async function stopBackgroundSharing(): Promise<void> {
  if (Platform.OS === 'web') return;
  try {
    if (await Location.hasStartedLocationUpdatesAsync(TASK_NAME)) {
      await Location.stopLocationUpdatesAsync(TASK_NAME);
    }
  } catch {
    // Already stopped or unavailable.
  }
}
