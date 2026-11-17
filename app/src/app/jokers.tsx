import { router } from 'expo-router';
import { ActivityIndicator, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import Button from '@/components/button';
import JokerList from '@/components/joker-list';
import { useT } from '@/i18n';
import { colors, dynamicStyles, styles } from '@/styles';
import { useActivateJoker } from '@/use-activate-joker';
import { useGameState } from '@/use-game-state';

// Presented as a modal over the game screen (see _layout.tsx).
export default function JokersScreen() {
  const t = useT();
  const insets = useSafeAreaInsets();
  const { token, state, error, show, fail } = useGameState();

  const use = useActivateJoker(token, show, fail);

  if (!token || !state) {
    return (
      <View style={[styles.screen, styles.centeredScreen]}>
        <ActivityIndicator color={colors.accent} />
      </View>
    );
  }
  return (
    <View style={styles.screen}>
      <View style={styles.flex}>
        <JokerList state={state} error={error} onUse={use} />
      </View>
      {/* Android cannot swipe a modal away, and iOS shows no grabber: an explicit way out. */}
      <View style={[styles.jokersFooterBase, dynamicStyles.jokersFooter(insets.bottom)]}>
        <Button label={t('joker.close')} variant="secondary" onPress={() => router.back()} />
      </View>
    </View>
  );
}
