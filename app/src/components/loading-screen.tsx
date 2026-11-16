import { ActivityIndicator, Text, View } from 'react-native';

import { colors, styles } from '@/styles';

/** Centered spinner, or the error that stopped the screen from loading. */
export default function LoadingScreen({ error }: { error: string }) {
  return (
    <View style={[styles.screen, styles.centeredScreen]}>
      {!!error && <Text style={[styles.text, styles.error]}>{error}</Text>}
      {!error && <ActivityIndicator color={colors.accent} />}
    </View>
  );
}
