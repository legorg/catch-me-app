import { View } from 'react-native';

import Button from '@/components/button';
import { useT } from '@/i18n';
import { styles } from '@/styles';

/** The ready toggle and the leave button. */
export default function LobbyActions({ ready, busy, readyDisabled, spectator, onToggleReady, onLeave }: {
  ready: boolean;
  busy: boolean;
  readyDisabled: boolean;
  spectator: boolean;
  onToggleReady: () => void;
  onLeave: () => void;
}) {
  const t = useT();
  return (
    <View style={styles.row}>
      {!spectator && <Button
        label={t(ready ? 'lobby.notReadyButton' : 'lobby.readyButton')}
        variant={ready ? 'secondary' : 'primary'}
        disabled={readyDisabled}
        onPress={onToggleReady}
      />}
      <Button label={t('lobby.leave')} variant="secondary" disabled={busy} onPress={onLeave} />
    </View>
  );
}
