import { useEffect, useState } from 'react';

import { isGameStateRunnerResponse, type GameStateResponse } from '@/api/types';
import { canShareGameLocation } from '@/game-state';
import { ensureNotificationPermission, notifyZoneAlerts, type NotificationPermission } from '@/zone-alert-notifications';

/**
 * While an active runner plays with the zone alert on: asks for the notification permission once
 * and notifies new alerts from every game state. Returns the permission, null until known.
 */
export function useZoneAlertNotifications(state: GameStateResponse | undefined): NotificationPermission | null {
  const wanted = !!state && isGameStateRunnerResponse(state) && state.jokerSettings.zoneAlert.enabled && canShareGameLocation(state);
  const [permission, setPermission] = useState<NotificationPermission | null>(null);

  useEffect(() => {
    if (!wanted) return;
    let current = true;
    void ensureNotificationPermission().then((result) => { if (current) setPermission(result); });
    return () => { current = false; };
  }, [wanted]);

  useEffect(() => {
    if (wanted && state) void notifyZoneAlerts(state);
  }, [wanted, state]);

  return wanted ? permission : null;
}
