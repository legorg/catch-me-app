import { useEffect, useState } from 'react';

import { isGameStateRunnerResponse, type GameStateResponse } from '@/api/types';
import { canShareGameLocation } from '@/game-state';
import { ensureNotificationPermission, notifyRunnerAlerts, type NotificationPermission } from '@/runner-alert-notifications';

/**
 * While a runner actively plays: asks for the notification permission once and notifies new
 * zone alerts and showRunners uses from every game state. Returns the permission, null until known.
 */
export function useRunnerAlertNotifications(state: GameStateResponse | undefined): NotificationPermission | null {
  const wanted = !!state && isGameStateRunnerResponse(state) && canShareGameLocation(state);
  const [permission, setPermission] = useState<NotificationPermission | null>(null);

  useEffect(() => {
    if (!wanted) return;
    let current = true;
    void ensureNotificationPermission().then((result) => { if (current) setPermission(result); });
    return () => { current = false; };
  }, [wanted]);

  useEffect(() => {
    if (wanted && state) void notifyRunnerAlerts(state);
  }, [wanted, state]);

  return wanted ? permission : null;
}
