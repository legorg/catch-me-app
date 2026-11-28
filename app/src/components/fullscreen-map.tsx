import { Ionicons } from '@expo/vector-icons';
import { StatusBar } from 'expo-status-bar';
import type { ReactNode } from 'react';
import { Modal, Pressable, ScrollView, View } from 'react-native';
import { SafeAreaProvider, SafeAreaView, useSafeAreaInsets } from 'react-native-safe-area-context';

import PlaygroundMap, { type PlaygroundMapProps } from './playground-map';
import { useT } from '../i18n';
import { colors, dynamicStyles, fullscreenStyles as styles, mapStyles } from '../styles';
import { useMapFullscreen } from '../use-map-fullscreen';

export type FullscreenMapProps = PlaygroundMapProps & {
  fullscreenFooter?: ReactNode;
  /** The inline map as a square as wide as its parent, instead of the default height. */
  square?: boolean;
};

/** The playground map with a fullscreen button and optional controls below the fullscreen map. */
export default function FullscreenMap({ fullscreenFooter, square = false, ...props }: FullscreenMapProps) {
  const t = useT();
  const { open, setOpen, inlineKey, close } = useMapFullscreen();
  return (
    <>
      <View style={square && mapStyles.square}>
        {/* Keep the space, but do not animate a second native map behind the modal. */}
        {open ? <View style={square ? mapStyles.fullscreen : mapStyles.map} />
          : <PlaygroundMap key={inlineKey} {...props} fullscreen={square || props.fullscreen} />}
        <Pressable accessibilityRole="button" accessibilityLabel={t('map.fullscreen')} onPress={() => setOpen(true)}
          style={[styles.toggle, dynamicStyles.fullscreenToggle({ top: 0, right: props.controlsInset?.right ?? 0 })]}>
          <Ionicons name="expand" size={22} color={colors.accent} />
        </Pressable>
      </View>
      <Modal visible={open} animationType="slide" onRequestClose={close}>
        {/* A modal is its own native window, so it needs its own provider for the insets. */}
        <SafeAreaProvider>
          <FullscreenContent {...props} fullscreenFooter={fullscreenFooter} onClose={close} />
        </SafeAreaProvider>
      </Modal>
    </>
  );
}

function FullscreenContent({ onClose, fullscreenFooter, ...props }: FullscreenMapProps & { onClose: () => void }) {
  const t = useT();
  const insets = useSafeAreaInsets();
  return (
    <View style={styles.full}>
      {/* Dark text over the light map; the layout's light style returns when the modal closes. */}
      <StatusBar style="dark" />
      <PlaygroundMap {...props} fullscreen />
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={t('map.close')}
        onPress={onClose}
        style={[styles.toggle, dynamicStyles.fullscreenToggle(insets)]}
      >
        <Ionicons name="contract" size={22} color={colors.accent} />
      </Pressable>
      {fullscreenFooter && (
        <SafeAreaView edges={['bottom', 'left', 'right']} style={styles.footer}>
          <ScrollView style={styles.footerScroll} contentContainerStyle={styles.footerContent}>
            {fullscreenFooter}
          </ScrollView>
        </SafeAreaView>
      )}
    </View>
  );
}
