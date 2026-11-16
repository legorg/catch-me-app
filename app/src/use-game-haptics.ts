import { useEffect, useRef } from 'react';
import { Vibration } from 'react-native';

import type { GameStateResponse } from '@/api/types';

/** Vibrates when the game starts and whenever a reveal happens. */
export function useGameHaptics(state: GameStateResponse | undefined) {
  // The game screen only opens when the countdown ends, so this buzzes once at the start.
  // (iOS ignores the pattern and vibrates once for a fixed time.)
  useEffect(() => {
    Vibration.vibrate([0, 400, 200, 400]);
  }, []);

  // A new nextRevealAt means the previous reveal just happened. The first value seen is not a reveal.
  const revealAt = state?.nextRevealAt;
  const lastRevealAt = useRef<string | undefined>(undefined);
  useEffect(() => {
    if (lastRevealAt.current && revealAt && revealAt !== lastRevealAt.current) Vibration.vibrate(300);
    if (revealAt) lastRevealAt.current = revealAt;
  }, [revealAt]);
}
