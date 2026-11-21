import { Pressable, Text, View } from 'react-native';

import type { Coordinates, Playground } from '@/api/types';
import FullscreenMap from '@/components/fullscreen-map';
import { useT } from '@/i18n';
import { ROUTE_COLORS, ROUTE_PIN_COLOR } from '@/lobby';
import { cellAt, routePointAt, rowCells } from '@/playground';
import { picker, styles } from '@/styles';
import type { RouteMode, RoutePoints } from '@/use-create-game';

const MODES: RouteMode[] = ['generated', 'chosen'];

/** Card for choosing whether the server generates the route or the creator taps start and stop on the map. */
export default function RouteChoiceCard({ mode, onModeChange, playground, points, tapMissed, onTap, disabled }: {
  mode: RouteMode;
  onModeChange: (mode: RouteMode) => void;
  playground: Playground | null;
  points: RoutePoints;
  tapMissed: boolean;
  onTap: (point: Coordinates) => void;
  disabled: boolean;
}) {
  const t = useT();
  return (
    <View style={styles.card}>
      <Text style={styles.heading}>{t('create.route')}</Text>
      <Text style={styles.hint}>{t('create.routeHint')}</Text>
      <View style={picker.row}>
        {MODES.map((value) => {
          const selected = mode === value;
          const label = t(value === 'generated' ? 'create.routeGenerated' : 'create.routeChosen');
          return (
            <Pressable
              key={value}
              accessibilityRole="button"
              accessibilityLabel={`${t('create.route')}: ${label}`}
              accessibilityState={{ selected, disabled }}
              disabled={disabled}
              onPress={() => onModeChange(value)}
              style={[picker.cell, picker.cellWide, selected && picker.selected]}>
              <Text style={[picker.label, selected && picker.selectedLabel]}>{label}</Text>
            </Pressable>
          );
        })}
      </View>
      {mode === 'chosen' && playground && <RoutePicker playground={playground} points={points} tapMissed={tapMissed} onTap={onTap} disabled={disabled} />}
    </View>
  );
}

function RoutePicker({ playground, points, tapMissed, onTap, disabled }: {
  playground: Playground;
  points: RoutePoints;
  tapMissed: boolean;
  onTap: (point: Coordinates) => void;
  disabled: boolean;
}) {
  const t = useT();
  const startTitle = t('lobby.start');
  const stopTitle = t('lobby.target');
  const highlights = [
    ...rowCells(playground, playground.rows).map((cell) => ({ cell, title: startTitle, color: ROUTE_COLORS.start })),
    ...rowCells(playground, 1).map((cell) => ({ cell, title: stopTitle, color: ROUTE_COLORS.target })),
  ];
  const markers = (['start', 'stop'] as const).flatMap((which) => {
    const point = points[which];
    if (!point) return [];
    const title = which === 'start' ? startTitle : stopTitle;
    return [{ id: `chosen-${which}-${point.latitude}-${point.longitude}`, coordinate: point, title, description: title, color: ROUTE_PIN_COLOR }];
  });
  // The map refits to the grid only when it is resized, so it remounts when the grid moves.
  const key = `${playground.center.latitude},${playground.center.longitude},${playground.rows},${playground.cellSizeMeters}`;
  return (
    <View style={styles.gapTiny}>
      <Text style={styles.hint}>{t('create.routeTap')}</Text>
      <View style={styles.mapContainer}>
        <FullscreenMap key={key} playground={playground} editable={false} onSelect={noop} onTap={disabled ? noop : onTap} highlights={highlights} markers={markers} />
      </View>
      {tapMissed && <Text style={[styles.text, styles.error]}>{t('create.routeTapEdge')}</Text>}
      <PointStatus playground={playground} point={points.start} which="start" />
      <PointStatus playground={playground} point={points.stop} which="stop" />
    </View>
  );
}

function PointStatus({ playground, point, which }: { playground: Playground; point: Coordinates | null; which: 'start' | 'stop' }) {
  const t = useT();
  const name = t(which === 'start' ? 'lobby.start' : 'lobby.target');
  const cell = point && routePointAt(playground, point) === which ? cellAt(playground, point) : null;
  if (!point) return <Text style={styles.hint}>{t('create.routePointMissing', { point: name })}</Text>;
  if (!cell) return <Text style={[styles.text, styles.error]}>{t('create.routePointMoved', { point: name })}</Text>;
  return (
    <Text style={[styles.text, which === 'start' ? styles.startText : styles.targetText]}>
      {t('create.routePoint', { point: name, cell })}
    </Text>
  );
}

const noop = () => {};
