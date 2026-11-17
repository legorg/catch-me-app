import { router } from 'expo-router';
import { ActivityIndicator, Platform, Text, TextInput, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import Button from '@/components/button';
import KeyboardScrollView from '@/components/keyboard-scroll-view';
import LanguageSwitch from '@/components/language-switch';
import NameInput from '@/components/name-input';
import { useT } from '@/i18n';
import { colors, dynamicStyles, languageSwitch, styles } from '@/styles';
import { useJoinGame } from '@/use-join-game';
import { useResumeGame } from '@/use-resume-game';

/** The title in place of the native header, with the language dropdown on the same line. */
function HomeTitle() {
  const insets = useSafeAreaInsets();
  return (
    <View style={[languageSwitch.titleRow, Platform.OS === 'android' && dynamicStyles.homeTitle(insets.top)]}>
      <Text accessibilityRole="header" style={languageSwitch.title}>Catch me!</Text>
      <LanguageSwitch />
    </View>
  );
}

export default function HomeScreen() {
  const t = useT();
  const { code, setCode, name, setName, busy, error, canJoin, join } = useJoinGame();
  const resuming = useResumeGame();

  if (resuming) {
    return <View style={styles.screen}><ActivityIndicator color={colors.accent} /></View>;
  }

  return (
    <>
      <KeyboardScrollView>
        <HomeTitle />
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
            keyboardType="number-pad"
            autoCorrect={false}
            maxLength={6}
            style={styles.input}
          />
          <NameInput value={name} onChange={setName} editable={!busy} />
          <Text style={styles.hint}>{t('home.codeHint')}</Text>
          {!!error && <Text accessibilityLiveRegion="polite" style={[styles.text, styles.error]}>{error}</Text>}
          {busy && <ActivityIndicator color={colors.accent} />}
          <Button label={t('home.join')} onPress={join} disabled={busy || !canJoin} />
        </View>
        <View style={styles.card}>
          <Text style={styles.heading}>{t('home.createHeading')}</Text>
          <Text style={styles.text}>{t('home.createText')}</Text>
          <Button label={t('home.create')} variant="secondary" onPress={() => router.push('/create')} disabled={busy} />
        </View>
        <Text style={styles.hint}>{t('home.devHint')}</Text>
      </KeyboardScrollView>
    </>
  );
}
