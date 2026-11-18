import { useRef, useState } from 'react';

import { ApiError, markCaught, type GameStateResponse } from '@/api/client';
import { confirm } from '@/confirm';
import { isActivePlayer, ownPlayer } from '@/game-state';
import { useT } from '@/i18n';

type Options = {
  token: string | null | undefined;
  state: GameStateResponse | undefined;
  show: (state: GameStateResponse) => void;
  fail: (err: unknown) => void;
};

/** Confirmation and request state for the irreversible, fairness-based catch. */
export function useMarkCaught({ token, state, show, fail }: Options) {
  const t = useT();
  const pending = useRef(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const submit = async () => {
    if (!token || state?.team !== 'runner' || state.status !== 'running' || pending.current || !isActivePlayer(ownPlayer(state))) return;
    pending.current = true;
    setBusy(true);
    setError('');
    try {
      show(await markCaught(token));
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) fail(err);
      else setError(err instanceof Error ? err.message : t('common.backendUnreachable'));
    } finally {
      pending.current = false;
      setBusy(false);
    }
  };

  const confirmCatch = () => confirm(t('catch.confirmTitle'), t('catch.confirmText'), t('catch.confirmYes'), () => void submit());
  return { busy, error, confirmCatch };
}
