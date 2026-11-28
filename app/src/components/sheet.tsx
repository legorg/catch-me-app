import type { ReactNode } from 'react';
import { Animated, Modal, Pressable, StyleSheet, Text, View } from 'react-native';
import { SafeAreaProvider, SafeAreaView } from 'react-native-safe-area-context';

import { noop } from '../press';
import { dynamicStyles, sheetStyles as styles } from '../styles';
import { useSheet } from '../use-sheet';

type Props = {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
};

/** A card that slides up over a fading, dimmed screen; a tap on the dimmed part or Android's back button closes it. */
export default function Sheet({ open, onClose, title, children }: Props) {
  const { visible, backdropOpacity, offset } = useSheet(open);
  return (
    <Modal visible={visible} transparent animationType="none" statusBarTranslucent onRequestClose={onClose}>
      {/* A modal is its own native window, so it needs its own provider for the insets. */}
      <SafeAreaProvider>
        <View style={styles.container}>
          <Animated.View style={[StyleSheet.absoluteFill, styles.backdrop, dynamicStyles.opacity(backdropOpacity)]}>
            <Pressable style={StyleSheet.absoluteFill} onPress={onClose} />
          </Animated.View>
          <Animated.View style={dynamicStyles.translateY(offset)}>
            {/* Swallows taps on the card, so only the dimmed part closes the sheet. */}
            <Pressable onPress={noop}>
              <SafeAreaView edges={['bottom', 'left', 'right']} style={styles.sheet}>
                <View style={styles.handle} />
                <Text style={styles.title}>{title}</Text>
                {children}
              </SafeAreaView>
            </Pressable>
          </Animated.View>
        </View>
      </SafeAreaProvider>
    </Modal>
  );
}
