import { Text } from 'react-native';

import { useCreateGameContext } from '@/create-game-context';
import { useT } from '@/i18n';
import { styles } from '@/styles';

/** What the create steps are doing or what went wrong; read out when it changes. */
export default function CreateMessage() {
  const t = useT();
  const { message, error } = useCreateGameContext();
  return (
    <Text accessibilityLiveRegion="polite" style={[styles.hint, error && styles.error]}>
      {'key' in message ? t(message.key) : message.text}
    </Text>
  );
}
