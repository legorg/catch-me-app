import { router } from 'expo-router';
import { useState } from 'react';

import { createGame, joinGame, type Coordinates, type CreateGameResponse } from '@/api/client';
import { useT, type MessageKey } from '@/i18n';
import { playerSession } from '@/player-session';
import { DEFAULT_CELL_SIZE_METERS, DEFAULT_CENTER, parseCellSizeKm, previewPlayground } from '@/playground';
import { useCurrentPosition } from '@/use-current-position';

// Own messages are kept as keys, so they follow a language switch; error texts are shown as received.
type Message = { key: MessageKey } | { text: string };
type Status = { message: Message; error: boolean; busy: boolean };

/** State and actions of the create-game screen: choosing the grid, creating the game, and joining it. */
export function useCreateGame() {
  const t = useT();
  // Where the device is, once known; the grid starts there instead of at the default spot.
  const here = useCurrentPosition();
  const [picked, setPicked] = useState<Coordinates | null>(null);
  const [cellSizeText, setCellSizeText] = useState(String(DEFAULT_CELL_SIZE_METERS / 1000));
  const [created, setCreated] = useState<CreateGameResponse | null>(null);
  const [name, setName] = useState('');
  const [status, setStatus] = useState<Status>({ message: { key: 'create.pickCenter' }, error: false, busy: false });
  const { busy } = status;
  const cellSize = parseCellSizeKm(cellSizeText);
  // Until the user picks a spot, the grid follows the device position.
  const selected = picked ?? here;

  let playground = created?.game.playground ?? null;
  let sizeError = !created && cellSize === null ? t('create.cellSizeInvalid') : '';
  // The grid starts at a default spot; the user drags it, and can only create once they have.
  if (!created && cellSize !== null) {
    try {
      playground = previewPlayground(selected ?? DEFAULT_CENTER, cellSize);
    } catch (err) {
      sizeError = err instanceof Error ? err.message : t('create.sizeInvalid');
    }
  }
  const displayedSize = created?.game.playground.cellSizeMeters ?? cellSize;
  const cannotCreate = !selected || !playground || !!sizeError;

  function fail(err: unknown, fallback: MessageKey) {
    setStatus((s) => ({ ...s, error: true, message: err instanceof Error ? { text: err.message } : { key: fallback } }));
  }

  function selectCenter(center: Coordinates) {
    try {
      if (cellSize === null) throw new Error(t('create.sizeFirst'));
      previewPlayground(center, cellSize);
      setPicked(center);
      setStatus((s) => ({ ...s, error: false, message: { key: 'create.checkGrid' } }));
    } catch (err) {
      fail(err, 'create.centerInvalid');
    }
  }

  async function run(action: () => Promise<void>, progress: MessageKey) {
    if (busy) return;
    setStatus({ busy: true, error: false, message: { key: progress } });
    try {
      await action();
    } catch (err) {
      fail(err, 'common.backendUnreachable');
    } finally {
      setStatus((s) => ({ ...s, busy: false }));
    }
  }

  const create = () => {
    void run(async () => {
      if (!selected || cellSize === null || cannotCreate) return;
      setCreated(await createGame(selected, cellSize));
      setStatus((s) => ({ ...s, message: { key: 'create.created' } }));
    }, 'create.creating');
  };

  const joinAs = (code: string) => {
    void run(async () => {
      const joined = await joinGame(code, name);
      playerSession.set(joined.playerToken);
      router.replace('/lobby');
    }, 'create.joining');
  };

  return {
    created, playground, displayedSize, hasPosition: !!here,
    cellSizeText, setCellSizeText, sizeError,
    name, setName, busy, message: status.message, error: status.error, cannotCreate,
    selectCenter, create, joinAs,
  };
}
