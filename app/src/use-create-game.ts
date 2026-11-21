import { router } from 'expo-router';
import { useState } from 'react';

import { createGame, joinGame, type Coordinates, type CreateGameResponse } from '@/api/client';
import type { JokerSettings } from '@/api/types';
import { useT, type MessageKey } from '@/i18n';
import { DEFAULT_JOKER_SETTINGS, withAllowance, withZoneAlert, type JokerKind } from '@/joker-settings';
import { playerSession } from '@/player-session';
import {
  DEFAULT_CELL_SIZE_METERS, DEFAULT_CENTER, DEFAULT_GRID_SIZE, DEFAULT_REVEAL_INTERVAL_MINUTES,
  isValidGridSize, isValidRevealInterval, parseCellSizeKm, parseWholeNumber, previewPlayground, rotationFromHeading, routePointAt,
} from '@/playground';
import { useCurrentPosition } from '@/use-current-position';

// Own messages are kept as keys, so they follow a language switch; error texts are shown as received.
type Message = { key: MessageKey } | { text: string };
type Status = { message: Message; error: boolean; busy: boolean };

/** Whether the server generates the route from landmarks or the creator places start and stop on the map. */
export type RouteMode = 'generated' | 'chosen';
export type RoutePoints = { start: Coordinates | null; stop: Coordinates | null };

/** State and actions of the create-game screen: choosing the grid, creating the game, and joining it. */
export function useCreateGame() {
  const t = useT();
  // Where the device is, once known; the grid starts there instead of at the default spot.
  const here = useCurrentPosition();
  const [picked, setPicked] = useState<Coordinates | null>(null);
  const [cellSizeText, setCellSizeText] = useState(String(DEFAULT_CELL_SIZE_METERS / 1000));
  const [gridSizeText, setGridSizeText] = useState(String(DEFAULT_GRID_SIZE));
  const [revealIntervalText, setRevealIntervalText] = useState(String(DEFAULT_REVEAL_INTERVAL_MINUTES));
  // Turning the map turns the grid with it, so the grid always looks straight on screen.
  const [rotation, setRotation] = useState(0);
  const [jokers, setJokers] = useState<JokerSettings>(DEFAULT_JOKER_SETTINGS);
  const [routeMode, setRouteMode] = useState<RouteMode>('generated');
  const [routePoints, setRoutePoints] = useState<RoutePoints>({ start: null, stop: null });
  // Set when the last tap on the route map was outside both edge rows.
  const [routeTapMissed, setRouteTapMissed] = useState(false);
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
      playground = previewPlayground(selected ?? DEFAULT_CENTER, cellSize, gridSize, rotation);
    } catch (err) {
      sizeError = err instanceof Error ? err.message : t('create.sizeInvalid');
    }
  }
  const displayedSize = created?.game.playground.cellSizeMeters ?? cellSize;
  const displayedGridSize = created?.game.playground.rows ?? gridSize;
  // Points stay where they were tapped, so moving or resizing the grid can leave them outside their edge row.
  const routeValid = !!playground && !!routePoints.start && !!routePoints.stop
    && routePointAt(playground, routePoints.start) === 'start' && routePointAt(playground, routePoints.stop) === 'stop';
  /** Why the game cannot be created yet, or null. */
  const createBlocker = sizeError || gridSizeError || revealIntervalError
    || (!selected || !playground ? t('create.pickCenter') : null)
    || (routeMode === 'chosen' && !routeValid ? t('create.routeTap') : null);
  const cannotCreate = createBlocker !== null;

  function fail(err: unknown, fallback: MessageKey) {
    setStatus((s) => ({ ...s, error: true, message: err instanceof Error ? { text: err.message } : { key: fallback } }));
  }

  function selectCenter(center: Coordinates) {
    try {
      if (cellSize === null || gridSize === null) throw new Error(t('create.sizeFirst'));
      previewPlayground(center, cellSize, gridSize, rotation);
      setPicked(center);
      setStatus((s) => ({ ...s, error: false, message: { key: 'create.checkGrid' } }));
    } catch (err) {
      fail(err, 'create.centerInvalid');
    }
  }

  /** The user turned the create map: the grid turns by the same whole degrees, until the game is created. */
  function turnGrid(heading: number) {
    if (!created) setRotation(rotationFromHeading(heading));
  }

  /** A tap on the route map sets the start (southern edge row) or the stop (northern edge row). */
  function tapRoutePoint(point: Coordinates) {
    const which = playground && routePointAt(playground, point);
    setRouteTapMissed(!which);
    if (which) setRoutePoints((current) => ({ ...current, [which]: point }));
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
      const { start, stop } = routePoints;
      const route = routeMode === 'chosen' && start && stop ? { start, stop } : undefined;
      setCreated(await createGame(selected, cellSize, gridSize, rotation, revealInterval, jokers, route));
      setStatus((s) => ({ ...s, message: { key: 'create.created' } }));
    }, routeMode === 'chosen' ? 'create.creatingChosen' : 'create.creating');
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
    rotation, turnGrid,
    jokers, setJokerAllowance: (kind: JokerKind, allowance: number) => setJokers((current) => withAllowance(current, kind, allowance)),
    setZoneAlert: (enabled: boolean) => setJokers((current) => withZoneAlert(current, enabled)),
    routeMode, setRouteMode, routePoints, routeTapMissed, tapRoutePoint,
    name, setName, busy, message: status.message, error: status.error, cannotCreate, createBlocker,
    selectCenter, create, joinAs,
  };
}
