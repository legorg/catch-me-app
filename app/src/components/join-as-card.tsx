import { Text, View } from 'react-native';

import { MAX_NAME_LENGTH, isValidPlayerName, type JoinCodes } from '@/api/client';
import Button from '@/components/button';
import NameInput from '@/components/name-input';
import { useT } from '@/i18n';
import { styles } from '@/styles';

/** The creator has the same playing/watching choices as any invited participant. */
export default function JoinAsCard({ codes, name, onNameChange, busy, onJoin }: {
  codes: JoinCodes;
  name: string;
  onNameChange: (name: string) => void;
  busy: boolean;
  onJoin: (code: string) => void;
}) {
  const t = useT();
  const nameValid = isValidPlayerName(name);
  const disabled = busy || !nameValid;
  const disabledReason = nameValid ? null : t('client.name', { max: MAX_NAME_LENGTH });
  return (
    <View style={styles.card}>
      <Text style={styles.heading}>{t('create.joinYourself')}</Text>
      <NameInput value={name} onChange={onNameChange} editable={!busy} />
      <View style={styles.row}>
        <Button label={t('create.joinAsRunner')} onPress={() => onJoin(codes.runner)} disabled={disabled} disabledReason={disabledReason} />
        <Button label={t('create.joinAsHunter')} onPress={() => onJoin(codes.hunter)} disabled={disabled} disabledReason={disabledReason} />
        <Button label={t('create.joinAsSpectator')} onPress={() => onJoin(codes.spectator)} disabled={disabled} disabledReason={disabledReason} />
      </View>
    </View>
  );
}
