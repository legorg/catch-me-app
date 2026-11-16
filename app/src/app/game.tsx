import { StatusBar } from 'expo-status-bar';
import { useState } from 'react';
import { StyleSheet, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { isGameStateHunterResponse } from '@/api/types';
import ActionBar from '@/components/action-bar';
import ErrorPills from '@/components/error-pills';
import GameHud from '@/components/game-hud';
import InfoPanel from '@/components/info-panel';
import LoadingScreen from '@/components/loading-screen';
import PlaygroundMap from '@/components/playground-map';
import { gameHighlights, gameMarkers, secondsUntil, sketchLines } from '@/lobby';
import { cellAt } from '@/playground';
import { dynamicStyles, overlay, styles } from '@/styles';
import { useGameHaptics } from '@/use-game-haptics';
import { useGameLocation } from '@/use-game-location';
import { useGameState, useLobbyUnlessRunning } from '@/use-game-state';
import { useSketching } from '@/use-sketching';

export default function GameScreen() {
  const insets = useSafeAreaInsets();
  const { token, snapshot, state, now, error, show, fail, running } = useGameState();
  const [infoOpen, setInfoOpen] = useState(false);
  const sketching = useSketching({ token, show, fail });
  const location = useGameLocation({ token, state, show, fail });
  useGameHaptics(state);
  useLobbyUnlessRunning(state, running);

  if (!token || !state || !snapshot) return <LoadingScreen error={error} />;

  const me = state.players.find((p) => p.playerId === state.playerId);
  const hunter = isGameStateHunterResponse(state) ? state : undefined;
  const canSketch = !!hunter && !!me?.hunterTeam;
  const drawing = canSketch && sketching.drawing;
  const outside = !!me?.position && cellAt(state.playground, me.position) === null;
  const nextReveal = secondsUntil(state.nextRevealAt, state, snapshot.receivedAt, now);

  const startDrawing = () => {
    sketching.setDrawing(true);
    setInfoOpen(false);
    sketching.clearError();
  };
  const stopDrawing = () => {
    sketching.setDrawing(false);
    sketching.clearError();
  };

  return (
    <View style={styles.screen}>
      {/* Dark text over the light map; the layout's light style returns when this screen closes. */}
      <StatusBar style="dark" />
      <View style={StyleSheet.absoluteFill}>
        <PlaygroundMap playground={state.playground} editable={false} onSelect={() => {}} highlights={gameHighlights(state)} markers={gameMarkers(state)} lines={hunter ? sketchLines(hunter) : []} onDraw={drawing ? sketching.send : undefined} fullscreen controlsInset={{ bottom: insets.bottom, right: insets.right }} />
      </View>
      {/* Floating over the map; touches outside the cards fall through to it. */}
      <View pointerEvents="box-none" style={[overlay.fill, dynamicStyles.hudFill(insets)]}>
        <GameHud nextReveal={nextReveal} outside={outside} drawing={drawing} />
        <View pointerEvents="box-none" style={styles.hudZone}>
          {infoOpen && <InfoPanel state={state} me={me} revealScheduled={nextReveal !== null} location={location} />}
          <ErrorPills location={location.error} poll={error} sketch={sketching.error} />
          <ActionBar
            drawing={drawing}
            canSketch={canSketch}
            hasSketches={!!hunter && hunter.sketches.length > 0}
            infoOpen={infoOpen}
            onToggleInfo={() => setInfoOpen(!infoOpen)}
            onDraw={startDrawing}
            onDoneDrawing={stopDrawing}
            onClearSketches={sketching.confirmClear}
          />
        </View>
      </View>
    </View>
  );
}
