import type { ReactNode } from 'react';
import { KeyboardAvoidingView, Platform, ScrollView } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { styles } from '@/styles';

/**
 * Screen content that shrinks above the open keyboard, so focused inputs stay visible.
 * Expo apps are edge-to-edge on Android, so the window itself no longer resizes.
 *
 * On iOS the ScrollView must be the screen's first native child, otherwise the large header
 * does not link to it and only expands after its first animation.
 */
export default function KeyboardScrollView({ children }: { children: ReactNode }) {
  const insets = useSafeAreaInsets();
  if (Platform.OS === 'ios') {
    return (
      <ScrollView
        style={{ flex: 1 }}
        contentInsetAdjustmentBehavior="automatic"
        automaticallyAdjustKeyboardInsets
        contentContainerStyle={styles.content}
        keyboardShouldPersistTaps="handled"
      >
        {children}
      </ScrollView>
    );
  }
  return (
    <KeyboardAvoidingView behavior="padding" style={{ flex: 1 }}>
      <ScrollView
        contentContainerStyle={[styles.content, { paddingBottom: styles.content.padding + insets.bottom }]}
        keyboardShouldPersistTaps="handled"
      >
        {children}
      </ScrollView>
    </KeyboardAvoidingView>
  );
}
