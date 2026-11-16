import { Modal, Text, View } from 'react-native';

import { useT } from '@/i18n';
import { noop } from '@/lobby';
import { styles } from '@/styles';

/** Covers everything until the game screen replaces the lobby; un-readying or leaving closes it. */
export default function LobbyCountdownModal({ countdown }: { countdown: number | null }) {
  const t = useT();
  return (
    <Modal visible={countdown !== null && countdown > 0} animationType="fade" statusBarTranslucent onRequestClose={noop}>
      <View style={[styles.screen, styles.countdownScreen]}>
        <Text style={styles.heading}>{t('lobby.countdownTitle')}</Text>
        <Text accessibilityLiveRegion="polite" style={styles.countdown}>{countdown}</Text>
        <Text style={[styles.hint, styles.centerText]}>{t('lobby.countdownHint')}</Text>
      </View>
    </Modal>
  );
}
