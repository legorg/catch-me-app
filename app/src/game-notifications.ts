import * as Notifications from 'expo-notifications';
import { AndroidImportance, IosAuthorizationStatus, SchedulableTriggerInputTypes } from 'expo-notifications';
import { Platform } from 'react-native';

import type { GameStateResponse } from './api/types';
import { translate } from './i18n';
import { alertNotifications, revealWarning } from './lobby';

const CHANNEL_ID = 'runner-alerts';

export type NotificationPermission = 'granted' | 'denied' | 'unsupported';

// Shared by the game screen and the background location task, which run in the same JavaScript runtime.
const notified = new Set<string>();
// The scheduled reveal warning as identifier and reveal time, so each new state only reschedules a changed one.
let scheduledWarning: { identifier: string; revealAt: string } | null = null;

// Must run at module load (imported via the background location task, which the root layout loads
// first): without a handler, notifications are not shown while the app is in the foreground.
if (Platform.OS !== 'web') {
  Notifications.setNotificationHandler({
    handleNotification: async () => ({ shouldShowBanner: true, shouldShowList: true, shouldPlaySound: true, shouldSetBadge: false }),
  });
}

/** Android 13+ only asks for the permission once a channel exists. Uses the language active at creation. */
async function ensureChannel() {
  if (Platform.OS !== 'android') return;
  await Notifications.setNotificationChannelAsync(CHANNEL_ID, {
    name: translate('notification.channel'),
    importance: AndroidImportance.HIGH,
    vibrationPattern: [0, 400, 200, 400],
  });
}

function allowed(permission: Notifications.NotificationPermissionsStatus): boolean {
  return permission.granted || permission.ios?.status === IosAuthorizationStatus.PROVISIONAL;
}

/** Asks once if the system still allows asking; a denial is final until the user changes the system settings. */
export async function ensureNotificationPermission(): Promise<NotificationPermission> {
  if (Platform.OS === 'web') return 'unsupported';
  try {
    await ensureChannel();
    const current = await Notifications.getPermissionsAsync();
    if (allowed(current)) return 'granted';
    if (!current.canAskAgain) return 'denied';
    return allowed(await Notifications.requestPermissionsAsync({ ios: { allowAlert: true, allowSound: true } })) ? 'granted' : 'denied';
  } catch {
    return 'denied';
  }
}

/**
 * Shows one local notification per new alert in the state and schedules the warning before the
 * next reveal (or cancels it once there is none); without permission the system drops them.
 * receivedAt is the device time the state arrived.
 */
export async function notifyGameAlerts(state: GameStateResponse, receivedAt = Date.now()): Promise<void> {
  if (Platform.OS === 'web') return;
  for (const { identifier, title, body } of alertNotifications(state, notified, Date.parse(state.serverTime))) {
    notified.add(identifier);
    try {
      await ensureChannel();
      await Notifications.scheduleNotificationAsync({
        identifier,
        content: { title, body },
        trigger: Platform.OS === 'android' ? { channelId: CHANNEL_ID } : null,
      });
    } catch {
      // The alert still shows in the app.
    }
  }
  const warning = revealWarning(state, receivedAt);
  if (!warning) {
    await cancelRevealWarning();
    return;
  }
  if (scheduledWarning?.identifier === warning.identifier && scheduledWarning.revealAt === warning.revealAt) return;
  scheduledWarning = { identifier: warning.identifier, revealAt: warning.revealAt };
  try {
    await ensureChannel();
    // The same identifier replaces a warning scheduled for an earlier reveal time.
    await Notifications.scheduleNotificationAsync({
      identifier: warning.identifier,
      content: { title: warning.title, body: warning.body },
      trigger: { type: SchedulableTriggerInputTypes.DATE, date: warning.fireAt, channelId: CHANNEL_ID },
    });
  } catch {
    scheduledWarning = null;
  }
}

/** Drops a pending reveal warning, e.g. once the runner is caught or finished, or the game ended. */
export async function cancelRevealWarning(): Promise<void> {
  if (Platform.OS === 'web' || !scheduledWarning) return;
  const { identifier } = scheduledWarning;
  scheduledWarning = null;
  try {
    await Notifications.cancelScheduledNotificationAsync(identifier);
  } catch {
    // Already shown or never scheduled.
  }
}
