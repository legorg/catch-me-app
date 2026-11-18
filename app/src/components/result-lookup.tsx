import { router } from 'expo-router';
import { useState } from 'react';
import { Text, TextInput, View } from 'react-native';

import Button from '@/components/button';
import { gameResultRoute, isValidGameId, normalizeGameId } from '@/game-result';
import { useT } from '@/i18n';
import { colors, styles } from '@/styles';

export default function ResultLookup({ disabled = false }: { disabled?: boolean }) {
  const t = useT();
  const [value, setValue] = useState('');
  const id = normalizeGameId(value);
  const valid = isValidGameId(id);
  return (
    <View style={styles.card}>
      <Text style={styles.heading}>{t('result.open')}</Text>
      <Text style={styles.text}>{t('result.lookupHint')}</Text>
      <TextInput
        accessibilityLabel={t('result.gameId')}
        placeholder={t('result.gameId')}
        placeholderTextColor={colors.placeholder}
        value={value}
        onChangeText={setValue}
        editable={!disabled}
        autoCapitalize="none"
        autoCorrect={false}
        style={styles.input}
      />
      {!!id && !valid && <Text style={[styles.hint, styles.error]}>{t('result.invalidId')}</Text>}
      <Button label={t('result.open')} variant="secondary" disabled={disabled || !valid} onPress={() => router.push(gameResultRoute(id))} />
    </View>
  );
}
