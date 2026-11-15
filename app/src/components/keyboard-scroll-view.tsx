import type { ReactNode } from 'react';
import { KeyboardAvoidingView, ScrollView } from 'react-native';

import { styles } from '@/styles';

/**
 * Screen content that shrinks above the open keyboard, so focused inputs stay visible.
 * Expo apps are edge-to-edge on Android, so the window itself no longer resizes.
 */
export default function KeyboardScrollView({ children }: { children: ReactNode }) {
  return (
    <KeyboardAvoidingView behavior="padding" style={{ flex: 1 }}>
      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        {children}
      </ScrollView>
    </KeyboardAvoidingView>
  );
}
