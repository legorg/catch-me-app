import { TextInput } from 'react-native';

import { MAX_NAME_LENGTH } from '@/api/client';
import { useT } from '@/i18n';
import { colors, styles } from '@/styles';

/** Text input for the player's name. */
export default function NameInput({ value, onChange, editable, testID }: {
  value: string;
  onChange: (name: string) => void;
  editable: boolean;
  testID?: string;
}) {
  const t = useT();
  return (
    <TextInput
      accessibilityLabel={t('common.yourName')}
      placeholder={t('common.yourName')}
      placeholderTextColor={colors.placeholder}
      value={value}
      onChangeText={onChange}
      editable={editable}
      autoCorrect={false}
      maxLength={MAX_NAME_LENGTH}
      style={styles.input}
      testID={testID}
    />
  );
}
