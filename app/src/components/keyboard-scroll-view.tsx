import type { ReactNode } from 'react';
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import Camouflage from '@/components/camouflage';
import { dynamicStyles, styles } from '@/styles';

type Props = {
  /** Scrolls along at the top, without the content's padding, e.g. a map from edge to edge. */
  top?: ReactNode;
  children: ReactNode;
};

/**
 * Screen content that shrinks above the open keyboard, so focused inputs stay visible.
 * Expo apps are edge-to-edge on Android, so the window itself no longer resizes.
 *
 * On iOS the ScrollView must be the screen's first native child, otherwise the large header
 * does not link to it and only expands after its first animation.
 */
export default function KeyboardScrollView({ top, children }: Props) {
  const insets = useSafeAreaInsets();
  if (Platform.OS === 'ios') {
    return (
      <ScrollView
        style={styles.flex}
        contentInsetAdjustmentBehavior="automatic"
        automaticallyAdjustKeyboardInsets
        keyboardShouldPersistTaps="handled"
      >
        <Camouflage style={styles.camouflage} />
        {top}
        <View style={styles.content}>{children}</View>
      </ScrollView>
    );
  }
  return (
    <KeyboardAvoidingView behavior="padding" style={styles.flex}>
      {/* Behind the ScrollView at screen size: Android tiles into one bitmap of the image's size,
          which fails for the oversized pattern iOS draws inside the scroll content. */}
      <Camouflage style={StyleSheet.absoluteFill} />
      <ScrollView keyboardShouldPersistTaps="handled">
        {top}
        <View style={[styles.content, dynamicStyles.scrollBottom(insets.bottom)]}>{children}</View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}
