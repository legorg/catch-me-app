import { Text, View } from 'react-native';

import { isValidPlayerName, type JoinCodes } from '@/api/client';
import Button from '@/components/button';
import NameInput from '@/components/name-input';
import { useT } from '@/i18n';
import { styles } from '@/styles';

/** Card for the game creator to join their own game as runner or hunter. */
export default function JoinAsCard({ codes, name, onNameChange, busy, onJoin }: {
  codes: JoinCodes;
  name: string;
  onNameChange: (name: string) => void;
  busy: boolean;
  onJoin: (code: string) => void;
}) {
  const t = useT();
  const disabled = busy || !isValidPlayerName(name);
  return (
    <View style={styles.card}>
      <Text style={styles.heading}>{t('create.joinYourself')}</Text>
      <NameInput value={name} onChange={onNameChange} editable={!busy} />
      <View style={styles.row}>
        <Button label={t('create.joinAsRunner')} onPress={() => onJoin(codes.runner)} disabled={disabled} />
        <Button label={t('create.joinAsHunter')} onPress={() => onJoin(codes.hunter)} disabled={disabled} />
      </View>
    </View>
  );
}
