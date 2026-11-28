import { router } from 'expo-router';

import Button from '@/components/button';
import CreateMessage from '@/components/create-message';
import MapScrollView from '@/components/map-scroll-view';
import Slider from '@/components/slider';
import { useCreateGameContext } from '@/create-game-context';
import { formatNumber, useT } from '@/i18n';
import { CELL_SIZE_STEP_METERS, MAX_CELL_SIZE_METERS, MAX_GRID_SIZE, MIN_CELL_SIZE_METERS, MIN_GRID_SIZE } from '@/playground';

/** Step 1: drag and turn the map to place the grid, size it with the sliders below. */
export default function CreateAreaScreen() {
  const t = useT();
  const game = useCreateGameContext();
  const { playground, gridSize, cellSize, busy } = game;
  return (
    // Remounts once when the position arrives, so the map opens there.
    <MapScrollView key={game.hasPosition ? 'here' : 'default'}
      map={{ playground, editable: !busy, onSelect: game.selectCenter, alignToGrid: true, onHeadingChange: game.turnGrid }}>
      <Slider title={t('create.gridSize')} label={t('create.gridSizeLabel')} valueText={`${gridSize} × ${gridSize}`}
        value={gridSize} min={MIN_GRID_SIZE} max={MAX_GRID_SIZE} step={1} onChange={game.setGridSize} disabled={busy} />
      <Slider title={t('create.cellSize')} label={t('create.cellSizeLabel')} valueText={formatNumber(cellSize / 1000, { min: 1, max: 1 })}
        value={cellSize} min={MIN_CELL_SIZE_METERS} max={MAX_CELL_SIZE_METERS} step={CELL_SIZE_STEP_METERS} onChange={game.setCellSize} disabled={busy} />
      <CreateMessage />
      <Button label={t('create.next')} onPress={() => router.push('/create/rules')} disabled={game.areaBlocker !== null} disabledReason={game.areaBlocker} />
    </MapScrollView>
  );
}
