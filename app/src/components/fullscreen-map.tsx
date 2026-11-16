import { Ionicons } from '@expo/vector-icons';
import { Modal, Pressable, View } from 'react-native';
import { SafeAreaProvider, useSafeAreaInsets } from 'react-native-safe-area-context';

import PlaygroundMap, { type PlaygroundMapProps } from './playground-map';
import { useT } from '../i18n';
import { colors, dynamicStyles, fullscreenStyles as styles } from '../styles';
import { useMapFullscreen } from '../use-map-fullscreen';

/** The playground map with a button that opens it fullscreen. */
export default function FullscreenMap(props: PlaygroundMapProps) {
  const t = useT();
  const { open, setOpen, inlineKey, close } = useMapFullscreen();
  return (
    <>
      <View>
        <PlaygroundMap key={inlineKey} {...props} />
        <Pressable accessibilityRole="button" accessibilityLabel={t('map.fullscreen')} onPress={() => setOpen(true)} style={[styles.toggle, dynamicStyles.fullscreenToggle()]}>
          <Ionicons name="expand" size={22} color={colors.accent} />
        </Pressable>
      </View>
      <Modal visible={open} animationType="slide" onRequestClose={close}>
        {/* A modal is its own native window, so it needs its own provider for the insets. */}
        <SafeAreaProvider>
          <FullscreenContent {...props} onClose={close} />
        </SafeAreaProvider>
      </Modal>
    </>
  );
}

function FullscreenContent({ onClose, ...props }: PlaygroundMapProps & { onClose: () => void }) {
  const t = useT();
  const insets = useSafeAreaInsets();
  return (
    <View style={styles.full}>
      <PlaygroundMap {...props} fullscreen />
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={t('map.close')}
        onPress={onClose}
        style={[styles.toggle, dynamicStyles.fullscreenToggle(insets)]}
      >
        <Ionicons name="contract" size={22} color={colors.accent} />
      </Pressable>
    </View>
  );
}

