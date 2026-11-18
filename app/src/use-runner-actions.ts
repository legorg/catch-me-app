import { useEffect, useRef, useState } from 'react';

import { ApiError, finish, markCaught, type GameStateResponse } from '@/api/client';
import { confirm } from '@/confirm';
import { canFinish, canShareGameLocation } from '@/game-state';
import { useT } from '@/i18n';

type Options = {
  token: string | null | undefined;
  state: GameStateResponse | undefined;
  receivedAt: number | undefined;
  show: (state: GameStateResponse) => void;
  fail: (err: unknown) => void;
};

type Action = 'catch' | 'finish';

/** Irreversible runner actions share one request lock, so catch and finish cannot race. */
export function useRunnerActions({ token, state, receivedAt, show, fail }: Options) {
  const t = useT();
  const latest = useRef({ token, state, receivedAt });
  useEffect(() => { latest.current = { token, state, receivedAt }; }, [token, state, receivedAt]);
  const pending = useRef(false);
  const [busy, setBusy] = useState<Action | null>(null);
  const [error, setError] = useState('');

  const submit = async (action: Action) => {
    const { token, state, receivedAt } = latest.current;
    if (!token || state?.team !== 'runner' || !canShareGameLocation(state) || pending.current) return;
    if (action === 'finish' && (receivedAt === undefined || !canFinish(state, receivedAt, Date.now()))) {
      setError(t('finish.notEligible'));
      return;
    }
    pending.current = true;
    setBusy(action);
    setError('');
    try {
      show(await (action === 'catch' ? markCaught(token) : finish(token)));
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) fail(err);
      else setError(err instanceof Error ? err.message : t('common.backendUnreachable'));
    } finally {
      pending.current = false;
      setBusy(null);
    }
  };

  const confirmCatch = () => confirm(t('catch.confirmTitle'), t('catch.confirmText'), t('catch.confirmYes'), () => void submit('catch'));
  const confirmFinish = () => confirm(t('finish.confirmTitle'), t('finish.confirmText'), t('finish.confirmYes'), () => void submit('finish'));
  return { busy, error, confirmCatch, confirmFinish };
}
