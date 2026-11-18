import { StatusBar } from 'expo-status-bar';
import { StyleSheet, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { isGameStateHunterResponse } from '@/api/types';
import ActionBar from '@/components/action-bar';
import ErrorPills from '@/components/error-pills';
import GameHud from '@/components/game-hud';
import InfoPanel from '@/components/info-panel';
import LoadingScreen from '@/components/loading-screen';
import PlaygroundMap from '@/components/playground-map';
import { compassLines, gameHighlights, gameMarkers, secondsUntil, sketchLines } from '@/lobby';
import { canFinish, isActivePlayer, ownPlayer } from '@/game-state';
import { cellAt } from '@/playground';
import { noop } from '@/press';
import { dynamicStyles, overlay, styles } from '@/styles';
import { useDrawingControls } from '@/use-drawing-controls';
import { useGameHaptics } from '@/use-game-haptics';
import { useGameLocation } from '@/use-game-location';
import { useGameState, useLobbyUnlessRunning } from '@/use-game-state';
import { useSketching } from '@/use-sketching';
import { useRunnerActions } from '@/use-runner-actions';

export default function GameScreen() {
  const insets = useSafeAreaInsets();
  const { token, snapshot, state, now, error, show, fail, running } = useGameState();
  const sketching = useSketching({ token, show, fail });
  const { infoOpen, toggleInfo, startDrawing, stopDrawing } = useDrawingControls(sketching);
  const location = useGameLocation({ token, state, show, fail });
  const runnerActions = useRunnerActions({ token, state, receivedAt: snapshot?.receivedAt, show, fail });
  useGameHaptics(state);
  useLobbyUnlessRunning(state, running);

  if (!token || !state || !snapshot) return <LoadingScreen error={error} />;

  const me = ownPlayer(state);
  const active = isActivePlayer(me);
  const hunter = isGameStateHunterResponse(state) ? state : undefined;
  const canSketch = !!hunter && !!me?.hunterTeam;
  const drawing = canSketch && sketching.drawing;
  const outside = !!me?.position && cellAt(state.playground, me.position) === null;
  const nextReveal = secondsUntil(state.nextRevealAt, state, snapshot.receivedAt, now);

  return (
    <View style={styles.screen}>
      {/* Dark text over the light map; the layout's light style returns when this screen closes. */}
      <StatusBar style="dark" />
      <View style={StyleSheet.absoluteFill}>
        <PlaygroundMap playground={state.playground} editable={false} onSelect={noop} highlights={gameHighlights(state)} markers={gameMarkers(state)} lines={hunter ? [...sketchLines(hunter), ...compassLines(hunter)] : []} onDraw={drawing ? sketching.send : undefined} locationEnabled={active} fullscreen controlsInset={{ bottom: insets.bottom, right: insets.right }} />
      </View>
      {/* Floating over the map; touches outside the cards fall through to it. */}
      <View pointerEvents="box-none" style={[overlay.fill, dynamicStyles.hudFill(insets)]}>
        <GameHud nextReveal={nextReveal} outside={outside} drawing={drawing} runnerStatus={me?.runner?.status} testMode={state.startPositionCheckDisabled === true} />
        <View pointerEvents="box-none" style={styles.hudZone}>
          {infoOpen && <InfoPanel state={state} me={me} revealScheduled={nextReveal !== null} location={location} />}
          <ErrorPills location={active ? location.error : ''} poll={error} sketch={sketching.error} runnerAction={active ? runnerActions.error : ''} />
          <ActionBar
            drawing={drawing}
            canSketch={canSketch}
            canUseJokers={active}
            hasSketches={!!hunter && hunter.sketches.length > 0}
            infoOpen={infoOpen}
            onToggleInfo={toggleInfo}
            onDraw={startDrawing}
            onDoneDrawing={stopDrawing}
            onClearSketches={sketching.confirmClear}
            catchAction={state.team === 'runner' && state.status === 'running' && active ? { busy: runnerActions.busy === 'catch', disabled: !!runnerActions.busy, onPress: runnerActions.confirmCatch } : undefined}
            finishAction={active && (canFinish(state, snapshot.receivedAt, now) || runnerActions.busy === 'finish') ? { busy: runnerActions.busy === 'finish', disabled: !!runnerActions.busy, onPress: runnerActions.confirmFinish } : undefined}
          />
        </View>
      </View>
    </View>
  );
}
