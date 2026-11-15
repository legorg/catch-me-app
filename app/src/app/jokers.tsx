import { router } from 'expo-router';
import { ActivityIndicator, View } from 'react-native';

import { activateJoker, type JokerKind } from '@/api/client';
import JokerList from '@/components/joker-list';
import { colors, styles } from '@/styles';
import { useGameState } from '@/use-game-state';

// Presented as a native form sheet over the game screen (see _layout.tsx).
export default function JokersScreen() {
  const { token, state, error, show, fail } = useGameState();

  const use = async (kind: JokerKind) => {
    if (!token) return;
    try {
      show(await activateJoker(token, kind));
      // The map behind shows the result: the game screen reloads the state when the sheet closes.
      router.back();
    } catch (err) {
      fail(err);
    }
  };

  if (!token || !state) {
    return (
      <View style={[styles.screen, { justifyContent: 'center', padding: 20 }]}>
        <ActivityIndicator color={colors.accent} />
      </View>
    );
  }
  return (
    <View style={styles.screen}>
      <JokerList state={state} error={error} onUse={use} />
    </View>
  );
}
