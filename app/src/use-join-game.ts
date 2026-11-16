import { router } from 'expo-router';
import { useState } from 'react';

import { JOIN_CODE_PATTERN, isValidPlayerName, joinGame, normalizeJoinCode } from '@/api/client';
import { useT } from '@/i18n';
import { playerSession } from '@/player-session';

/** State and action of the home screen: entering a join code and name, and joining the game. */
export function useJoinGame() {
  const t = useT();
  const [code, setCode] = useState('');
  const [name, setName] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const canJoin = JOIN_CODE_PATTERN.test(normalizeJoinCode(code)) && isValidPlayerName(name);

  async function submit() {
    if (busy || !canJoin) return;
    setBusy(true);
    setError('');
    try {
      const joined = await joinGame(code, name);
      playerSession.set(joined.playerToken);
      router.replace('/lobby');
    } catch (err) {
      setError(err instanceof Error ? err.message : t('home.joinFailed'));
    } finally {
      setBusy(false);
    }
  }

  const join = () => {
    void submit();
  };

  return { code, setCode, name, setName, busy, error, canJoin, join };
}
