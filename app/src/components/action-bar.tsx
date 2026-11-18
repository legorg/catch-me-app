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

type RunnerAction = { busy: boolean; disabled: boolean; onPress: () => void };
/** Present for spectators only. */
type TimelineToggle = { open: boolean; onToggle: () => void };
type HudActionsProps = { canSketch: boolean; canUseJokers: boolean; infoOpen: boolean; onToggleInfo: () => void; onDraw: () => void; catchAction?: RunnerAction; finishAction?: RunnerAction; timeline?: TimelineToggle };

function HudActions({ canSketch, canUseJokers, infoOpen, onToggleInfo, onDraw, catchAction, finishAction, timeline }: HudActionsProps) {
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
      {timeline && <HudIconButton
        icon={timeline.open ? 'close' : 'list-outline'}
        size={22}
        label={t(timeline.open ? 'timeline.close' : 'timeline.open')}
        accessibilityState={{ expanded: timeline.open }}
        onPress={timeline.onToggle}
      />}
      {canUseJokers && <HudIconButton icon="flash-outline" size={22} label={t('joker.button')} onPress={() => router.push('/jokers')} />}
      {catchAction && <HudIconButton icon="hand-left-outline" size={22} label={t(catchAction.busy ? 'catch.sending' : 'catch.button')} disabled={catchAction.disabled} accessibilityState={{ busy: catchAction.busy }} onPress={catchAction.onPress} />}
      {finishAction && <HudIconButton icon="flag-outline" size={22} label={t(finishAction.busy ? 'finish.sending' : 'finish.button')} disabled={finishAction.disabled} accessibilityState={{ busy: finishAction.busy }} onPress={finishAction.onPress} />}
      {canSketch && <HudIconButton icon="brush-outline" size={22} label={t('sketch.draw')} onPress={onDraw} />}
    </View>
  );
}

type ActionBarProps = {
  drawing: boolean;
  canSketch: boolean;
  canUseJokers: boolean;
  hasSketches: boolean;
  infoOpen: boolean;
  onToggleInfo: () => void;
  onDraw: () => void;
  onDoneDrawing: () => void;
  onClearSketches: () => void;
  catchAction?: RunnerAction;
  finishAction?: RunnerAction;
  timeline?: TimelineToggle;
};

/** Bottom bar: the drawing toolbar while drawing, the normal buttons otherwise. */
export default function ActionBar({ drawing, canSketch, canUseJokers, hasSketches, infoOpen, onToggleInfo, onDraw, onDoneDrawing, onClearSketches, catchAction, finishAction, timeline }: ActionBarProps) {
  if (drawing) return <DrawingBar hasSketches={hasSketches} onDone={onDoneDrawing} onClear={onClearSketches} />;
  return <HudActions canSketch={canSketch} canUseJokers={canUseJokers} infoOpen={infoOpen} onToggleInfo={onToggleInfo} onDraw={onDraw} catchAction={catchAction} finishAction={finishAction} timeline={timeline} />;
}
