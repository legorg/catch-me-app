import { router } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { useState } from 'react';
import { ActivityIndicator, ScrollView, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { JOIN_CODE_PATTERN, MAX_NAME_LENGTH, isValidPlayerName, joinGame, normalizeJoinCode } from '@/api/client';
import Button from '@/components/button';
import LanguageSwitch from '@/components/language-switch';
import { useT } from '@/i18n';
import { playerSession } from '@/player-session';
import { colors, styles } from '@/styles';

export default function HomeScreen() {
  const t = useT();
  const [code, setCode] = useState('');
  const [name, setName] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const canJoin = JOIN_CODE_PATTERN.test(normalizeJoinCode(code)) && isValidPlayerName(name);

  async function join() {
    if (busy || !canJoin) return;
    setBusy(true);
    setError('');
    try {
      const joined = await joinGame(code, name);
      playerSession.set(joined.playerToken);
      router.replace('/lobby');
    } catch (err) {
      setError(err instanceof Error ? err.message : t('home.joinFailed'));
    } finally {
      setBusy(false);
    }
  }

  return (
    <SafeAreaView style={styles.screen}>
      <StatusBar style="light" />
      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        <LanguageSwitch />
        <Text style={styles.title}>Catch me!</Text>
        <Text style={styles.text}>{t('home.intro')}</Text>
        <View style={styles.card}>
          <Text style={styles.heading}>{t('home.joinHeading')}</Text>
          <TextInput
            accessibilityLabel={t('home.joinCode')}
            placeholder={t('home.joinCode')}
            placeholderTextColor={colors.placeholder}
            value={code}
            onChangeText={setCode}
            editable={!busy}
            autoCapitalize="characters"
            autoCorrect={false}
            maxLength={8}
            style={styles.input}
          />
          <TextInput
            accessibilityLabel={t('common.yourName')}
            placeholder={t('common.yourName')}
            placeholderTextColor={colors.placeholder}
            value={name}
            onChangeText={setName}
            editable={!busy}
            autoCorrect={false}
            maxLength={MAX_NAME_LENGTH}
            style={styles.input}
          />
          <Text style={styles.hint}>{t('home.codeHint')}</Text>
          {!!error && <Text accessibilityLiveRegion="polite" style={[styles.text, styles.error]}>{error}</Text>}
          {busy && <ActivityIndicator color={colors.accent} />}
          <Button label={t('home.join')} onPress={() => void join()} disabled={busy || !canJoin} />
        </View>
        <View style={styles.card}>
          <Text style={styles.heading}>{t('home.createHeading')}</Text>
          <Text style={styles.text}>{t('home.createText')}</Text>
          <Button label={t('home.create')} variant="secondary" onPress={() => router.push('/create')} disabled={busy} />
        </View>
        <Text style={styles.hint}>{t('home.devHint')}</Text>
      </ScrollView>
    </SafeAreaView>
  );
}
