import * as Notifications from 'expo-notifications';
import { AndroidImportance, IosAuthorizationStatus } from 'expo-notifications';
import { Platform } from 'react-native';

import type { GameStateResponse } from './api/types';
import { translate } from './i18n';
import { zoneAlertNotifications } from './lobby';

const CHANNEL_ID = 'zone-alerts';

export type NotificationPermission = 'granted' | 'denied' | 'unsupported';

// Shared by the game screen and the background location task, which run in the same JavaScript runtime.
const notified = new Set<string>();

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

/** Shows one local notification per new zone alert in the state; without permission the system drops it. */
export async function notifyZoneAlerts(state: GameStateResponse): Promise<void> {
  if (Platform.OS === 'web') return;
  for (const { identifier, body } of zoneAlertNotifications(state, notified, Date.parse(state.serverTime))) {
    notified.add(identifier);
    try {
      await ensureChannel();
      await Notifications.scheduleNotificationAsync({
        identifier,
        content: { title: translate('joker.zoneAlert.title'), body },
        trigger: Platform.OS === 'android' ? { channelId: CHANNEL_ID } : null,
      });
    } catch {
      // The alert still shows in the app.
    }
  }
}
