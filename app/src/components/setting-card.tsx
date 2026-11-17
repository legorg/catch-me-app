import { Text, TextInput, View, type KeyboardTypeOptions } from 'react-native';

import { styles } from '@/styles';

/** Card for entering one game setting as text, with a hint and its validation error. */
export default function SettingCard({ title, label, hint, value, onChange, editable, error, keyboardType = 'number-pad', maxLength = 3 }: {
  title: string;
  label: string;
  hint: string;
  value: string;
  onChange: (text: string) => void;
  editable: boolean;
  error: string;
  keyboardType?: KeyboardTypeOptions;
  maxLength?: number;
}) {
  return (
    <View style={styles.card}>
      <Text style={styles.heading}>{title}</Text>
      <TextInput
        accessibilityLabel={label}
        value={value}
        onChangeText={onChange}
        editable={editable}
        keyboardType={keyboardType}
        autoCorrect={false}
        maxLength={maxLength}
        style={styles.input}
      />
      <Text style={styles.hint}>{hint}</Text>
      {!!error && <Text accessibilityLiveRegion="polite" style={[styles.text, styles.error]}>{error}</Text>}
    </View>
  );
}
