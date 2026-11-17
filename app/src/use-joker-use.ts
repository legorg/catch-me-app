import { useState } from 'react';

import type { JokerKind, JokerRequest } from '@/api/client';

/** The request to use a joker, or null while its target is still missing. */
function jokerRequest(kind: JokerKind, cell: string | null): JokerRequest | null {
  switch (kind) {
    case 'queryZone':
      return cell ? { kind, cell } : null;
    case 'fakePosition':
      // The app cannot pick the point yet.
      return null;
    default:
      return { kind };
  }
}

/** The two-step flow of using a joker: ask for confirmation (and a cell for the zone joker), then use it. */
export function useJokerUse(onUse: (joker: JokerRequest) => Promise<void>) {
  const [confirming, setConfirming] = useState<JokerKind | null>(null);
  const [cell, setCell] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const use = async (kind: JokerKind) => {
    const joker = jokerRequest(kind, cell);
    if (busy || !joker) return;
    setBusy(true);
    try {
      await onUse(joker);
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
  /** Whether the joker has everything it needs to be used. */
  const ready = (kind: JokerKind) => jokerRequest(kind, cell) !== null;

  return { confirming, cell, setCell, busy, ready, use: (kind: JokerKind) => void use(kind), cancel, start };
}
