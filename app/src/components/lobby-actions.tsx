import { View } from 'react-native';

import Button from '@/components/button';
import { useT } from '@/i18n';
import { styles } from '@/styles';

/** The ready toggle and the leave button. */
export default function LobbyActions({ ready, busy, readyDisabled, onToggleReady, onLeave }: {
  ready: boolean;
  busy: boolean;
  readyDisabled: boolean;
  onToggleReady: () => void;
  onLeave: () => void;
}) {
  const t = useT();
  return (
    <View style={styles.row}>
      <Button
        label={t(ready ? 'lobby.notReadyButton' : 'lobby.readyButton')}
        variant={ready ? 'secondary' : 'primary'}
        disabled={readyDisabled}
        onPress={onToggleReady}
      />
      <Button label={t('lobby.leave')} variant="secondary" disabled={busy} onPress={onLeave} />
    </View>
  );
}
