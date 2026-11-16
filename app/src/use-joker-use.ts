import { useState } from 'react';

import type { JokerKind } from '@/api/client';

/** The two-step flow of using a joker: ask for confirmation (and a cell for the zone joker), then use it. */
export function useJokerUse(onUse: (kind: JokerKind, cell?: string) => Promise<void>) {
  const [confirming, setConfirming] = useState<JokerKind | null>(null);
  const [cell, setCell] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const use = async (kind: JokerKind) => {
    if (busy) return;
    setBusy(true);
    try {
      await onUse(kind, kind === 'queryZone' ? cell ?? undefined : undefined);
    } finally {
      setBusy(false);
      setConfirming(null);
      setCell(null);
    }
  };
  const cancel = () => {
    setConfirming(null);
    setCell(null);
  };
  const start = (kind: JokerKind) => {
    setConfirming(kind);
    setCell(null);
  };

  return { confirming, cell, setCell, busy, use: (kind: JokerKind) => void use(kind), cancel, start };
}
