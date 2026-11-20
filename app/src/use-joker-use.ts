import { useState } from 'react';

import type { JokerKind, JokerRequest } from '@/api/client';
import type { Coordinates, Playground } from '@/api/types';
import { cellAt } from '@/playground';

/** The request to use a joker, or null while its target is still missing (or the point is outside the playground). */
function jokerRequest(kind: JokerKind, cell: string | null, point: Coordinates | null, playground: Playground): JokerRequest | null {
  switch (kind) {
    case 'queryZone':
      return cell ? { kind, cell } : null;
    case 'fakePosition':
    case 'trap':
      return point && cellAt(playground, point) ? { kind, ...point } : null;
    default:
      return { kind };
  }
}

/** The two-step flow of using a joker: ask for confirmation (and a cell or point for some jokers), then use it. */
export function useJokerUse(playground: Playground, onUse: (joker: JokerRequest) => Promise<void>) {
  const [confirming, setConfirming] = useState<JokerKind | null>(null);
  const [cell, setCell] = useState<string | null>(null);
  const [point, setPoint] = useState<Coordinates | null>(null);
  const [busy, setBusy] = useState(false);

  const reset = () => {
    setCell(null);
    setPoint(null);
  };
  const use = async (kind: JokerKind) => {
    const joker = jokerRequest(kind, cell, point, playground);
    if (busy || !joker) return;
    setBusy(true);
    try {
      await onUse(joker);
    } finally {
      setBusy(false);
      setConfirming(null);
      reset();
    }
  };
  const cancel = () => {
    setConfirming(null);
    reset();
  };
  const start = (kind: JokerKind) => {
    setConfirming(kind);
    reset();
  };
  /** Whether the joker has everything it needs to be used. */
  const ready = (kind: JokerKind) => jokerRequest(kind, cell, point, playground) !== null;

  return { confirming, cell, setCell, point, setPoint, busy, ready, use: (kind: JokerKind) => void use(kind), cancel, start };
}
