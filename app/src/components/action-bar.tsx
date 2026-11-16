import { Pressable, Text, View } from 'react-native';
import { router } from 'expo-router';

import HudIconButton from '@/components/hud-icon-button';
import { useT } from '@/i18n';
import { overlay, styles } from '@/styles';

type DrawingBarProps = { hasSketches: boolean; onDone: () => void; onClear: () => void };

/** While drawing, a toolbar of its own replaces the normal buttons. */
function DrawingBar({ hasSketches, onDone, onClear }: DrawingBarProps) {
  const t = useT();
  return (
    <View style={overlay.bar}>
      <Pressable accessibilityRole="button" onPress={onDone} style={overlay.pill}>
        <Text style={styles.secondaryButtonText}>{t('sketch.done')}</Text>
      </Pressable>
      {hasSketches && <Pressable accessibilityRole="button" onPress={onClear} style={overlay.pill}>
        <Text style={[styles.secondaryButtonText, styles.error]}>{t('sketch.clear')}</Text>
      </Pressable>}
    </View>
  );
}

type HudActionsProps = { canSketch: boolean; infoOpen: boolean; onToggleInfo: () => void; onDraw: () => void };

function HudActions({ canSketch, infoOpen, onToggleInfo, onDraw }: HudActionsProps) {
  const t = useT();
  return (
    <View style={overlay.bar}>
      <HudIconButton
        icon={infoOpen ? 'close' : 'information-circle-outline'}
        size={24}
        label={t(infoOpen ? 'game.closeInfo' : 'game.info')}
        accessibilityState={{ expanded: infoOpen }}
        onPress={onToggleInfo}
      />
      <HudIconButton icon="flash-outline" size={22} label={t('joker.button')} onPress={() => router.push('/jokers')} />
      {canSketch && <HudIconButton icon="brush-outline" size={22} label={t('sketch.draw')} onPress={onDraw} />}
    </View>
  );
}

type ActionBarProps = {
  drawing: boolean;
  canSketch: boolean;
  hasSketches: boolean;
  infoOpen: boolean;
  onToggleInfo: () => void;
  onDraw: () => void;
  onDoneDrawing: () => void;
  onClearSketches: () => void;
};

/** Bottom bar: the drawing toolbar while drawing, the normal buttons otherwise. */
export default function ActionBar({ drawing, canSketch, hasSketches, infoOpen, onToggleInfo, onDraw, onDoneDrawing, onClearSketches }: ActionBarProps) {
  if (drawing) return <DrawingBar hasSketches={hasSketches} onDone={onDoneDrawing} onClear={onClearSketches} />;
  return <HudActions canSketch={canSketch} infoOpen={infoOpen} onToggleInfo={onToggleInfo} onDraw={onDraw} />;
}
