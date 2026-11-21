import { useEffect, useState } from 'react';

import type { GameStateResponse } from '@/api/types';
import { canShareGameLocation } from '@/game-state';
import { cancelRevealWarning, ensureNotificationPermission, notifyGameAlerts, type NotificationPermission } from '@/game-notifications';

/**
 * While a runner or hunter actively plays: asks for the notification permission once, notifies new
 * alerts from every game state and keeps the reveal warning scheduled. Returns the permission,
 * null until known.
 */
export function useGameNotifications(state: GameStateResponse | undefined, receivedAt: number): NotificationPermission | null {
  const wanted = canShareGameLocation(state);
  const [permission, setPermission] = useState<NotificationPermission | null>(null);

  useEffect(() => {
    if (!wanted) return;
    let current = true;
    void ensureNotificationPermission().then((result) => { if (current) setPermission(result); });
    return () => { current = false; };
  }, [wanted]);

  useEffect(() => {
    if (wanted && state) void notifyGameAlerts(state, receivedAt);
    // A caught or finished runner, or an ended game, gets no warning before a reveal that no longer concerns them.
    else void cancelRevealWarning();
  }, [wanted, state, receivedAt]);

  return wanted ? permission : null;
}
