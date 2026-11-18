import { useRef, useState } from 'react';
import { Share } from 'react-native';

import type { JoinCodes } from '@/api/types';
import { useT } from '@/i18n';
import { joinInvitationMessage } from '@/lobby';

/** Native sharing is optional; the selectable code remains available on failure/cancel. */
export function useShareJoinCode(codes: JoinCodes) {
  const t = useT();
  const pending = useRef(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const share = async (team: keyof JoinCodes) => {
    if (pending.current) return;
    pending.current = true;
    setBusy(true);
    setError('');
    try {
      await Share.share({ message: joinInvitationMessage(codes, team) });
    } catch {
      setError(t('joinCodes.shareFailed'));
    } finally {
      pending.current = false;
      setBusy(false);
    }
  };

  return { busy, error, share };
}
