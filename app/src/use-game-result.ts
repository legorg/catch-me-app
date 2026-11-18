import { router, useFocusEffect } from 'expo-router';
import { useCallback, useState } from 'react';

import { getGameResult, type GameResultResponse } from '@/api/client';

type Snapshot = { key: string; result?: GameResultResponse; error?: unknown };

export function returnFromResult() {
  // A directly opened result may be the only screen in the stack.
  if (router.canDismiss()) router.dismissAll();
  router.replace('/');
}

/** One read per focus/retry; old requests cannot replace a different game's result. */
export function useGameResult(sessionId: string) {
  const [snapshot, setSnapshot] = useState<Snapshot>({ key: '' });
  const [attempt, setAttempt] = useState(0);
  const key = `${sessionId}:${attempt}`;
  useFocusEffect(useCallback(() => {
    let active = true;
    setSnapshot({ key });
    void getGameResult(sessionId).then(
      (result) => { if (active) setSnapshot({ key, result }); },
      (error: unknown) => { if (active) setSnapshot({ key, error }); },
    );
    return () => { active = false; };
  }, [sessionId, key]));

  const current = snapshot.key === key ? snapshot : undefined;
  return { result: current?.result, error: current?.error, busy: !current?.result && !current?.error, retry: () => setAttempt((value) => value + 1) };
}
