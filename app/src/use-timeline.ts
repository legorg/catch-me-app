import { useFocusEffect } from 'expo-router';
import { useCallback, useState } from 'react';

import { getTimeline, type TimelineResponse } from '@/api/client';
import { translate } from '@/i18n';
import { newestFirst, shouldAcceptTimeline } from '@/timeline';

const POLL_INTERVAL_MS = 5000;

/**
 * Polls the spectator timeline every 5 seconds, only while it is open and the
 * screen is focused. An unknown token is handled by the game state poll.
 */
export function useTimeline(token: string | null, open: boolean) {
  const [timeline, setTimeline] = useState<TimelineResponse | null>(null);
  const [error, setError] = useState('');

  useFocusEffect(useCallback(() => {
    if (!token || !open) return;
    let active = true;
    const poll = () => getTimeline(token).then(
      (incoming) => {
        if (!active) return;
        setTimeline((previous) => shouldAcceptTimeline(previous, incoming) ? incoming : previous);
        setError('');
      },
      (err: unknown) => { if (active) setError(err instanceof Error ? err.message : translate('common.backendUnreachable')); },
    );
    void poll();
    const timer = setInterval(() => void poll(), POLL_INTERVAL_MS);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [token, open]));

  return { events: timeline ? newestFirst(timeline.events) : [], loaded: timeline !== null, error: open ? error : '' };
}
