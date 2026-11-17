import { router } from 'expo-router';
import { useState } from 'react';

import { createGame, joinGame, type Coordinates, type CreateGameResponse } from '@/api/client';
import { useT, type MessageKey } from '@/i18n';
import { playerSession } from '@/player-session';
import {
  DEFAULT_CELL_SIZE_METERS, DEFAULT_CENTER, DEFAULT_GRID_SIZE, DEFAULT_REVEAL_INTERVAL_MINUTES,
  isValidGridSize, isValidRevealInterval, parseCellSizeKm, parseWholeNumber, previewPlayground,
} from '@/playground';
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
  const [gridSizeText, setGridSizeText] = useState(String(DEFAULT_GRID_SIZE));
  const [revealIntervalText, setRevealIntervalText] = useState(String(DEFAULT_REVEAL_INTERVAL_MINUTES));
  const [created, setCreated] = useState<CreateGameResponse | null>(null);
  const [name, setName] = useState('');
  const [status, setStatus] = useState<Status>({ message: { key: 'create.pickCenter' }, error: false, busy: false });
  const { busy } = status;
  const cellSize = parseCellSizeKm(cellSizeText);
  const gridSize = parseWholeNumber(gridSizeText, isValidGridSize);
  const revealInterval = parseWholeNumber(revealIntervalText, isValidRevealInterval);
  // Until the user picks a spot, the grid follows the device position.
  const selected = picked ?? here;

  let playground = created?.game.playground ?? null;
  let sizeError = !created && cellSize === null ? t('create.cellSizeInvalid') : '';
  const gridSizeError = !created && gridSize === null ? t('create.gridSizeInvalid') : '';
  const revealIntervalError = !created && revealInterval === null ? t('create.revealIntervalInvalid') : '';
  // The grid starts at a default spot; the user drags it, and can only create once they have.
  if (!created && cellSize !== null && gridSize !== null) {
    try {
      playground = previewPlayground(selected ?? DEFAULT_CENTER, cellSize, gridSize);
    } catch (err) {
      sizeError = err instanceof Error ? err.message : t('create.sizeInvalid');
    }
  }
  const displayedSize = created?.game.playground.cellSizeMeters ?? cellSize;
  const displayedGridSize = created?.game.playground.rows ?? gridSize;
  const cannotCreate = !selected || !playground || !!sizeError || !!gridSizeError || !!revealIntervalError;

  function fail(err: unknown, fallback: MessageKey) {
    setStatus((s) => ({ ...s, error: true, message: err instanceof Error ? { text: err.message } : { key: fallback } }));
  }

  function selectCenter(center: Coordinates) {
    try {
      if (cellSize === null || gridSize === null) throw new Error(t('create.sizeFirst'));
      previewPlayground(center, cellSize, gridSize);
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
      if (!selected || cellSize === null || gridSize === null || revealInterval === null || cannotCreate) return;
      setCreated(await createGame(selected, cellSize, gridSize, revealInterval));
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
    created, playground, displayedSize, displayedGridSize, hasPosition: !!here,
    cellSizeText, setCellSizeText, sizeError,
    gridSizeText, setGridSizeText, gridSizeError,
    revealIntervalText, setRevealIntervalText, revealIntervalError,
    name, setName, busy, message: status.message, error: status.error, cannotCreate,
    selectCenter, create, joinAs,
  };
}
