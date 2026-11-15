import { getLocales } from 'expo-localization';
import { Stack } from 'expo-router';

// Registers the background location task before any screen renders.
import '@/background-location';
import LanguageSwitch from '@/components/language-switch';
import { isLanguage, setLanguage, useT } from '@/i18n';
import { colors } from '@/styles';

// Start in the first device language the app supports, otherwise English.
const deviceLanguage = getLocales().map((locale) => locale.languageCode).find(isLanguage);
if (deviceLanguage) setLanguage(deviceLanguage);

export default function RootLayout() {
  const t = useT();
  return (
    <Stack
      screenOptions={{
        statusBarStyle: 'light',
        headerBlurEffect: 'dark',
        headerTintColor: '#ffffff',
        headerBackButtonDisplayMode: 'minimal',
        headerTitleStyle: { fontWeight: '700' },
        headerShadowVisible: false,
        headerLargeTitle: true,
        headerLargeTitleShadowVisible: false,
        headerLargeTitleStyle: { color: '#ffffff' },
        contentStyle: { backgroundColor: colors.background },
       // headerRight: () => <LanguageSwitch />,
      }}
    >
      <Stack.Screen name="index" options={{ title: 'Catch me!' }} />
      <Stack.Screen name="create" options={{ title: t('create.chooseTitle') }} />
      {/* Joining replaces the previous screen, so there is nothing to go back to. */}
      <Stack.Screen name="lobby" options={{ title: t('lobby.title'), headerBackVisible: false, gestureEnabled: false }} />
      {/* The map needs the whole screen, so no large title; leaving goes through the exit button. */}
      <Stack.Screen name="game" options={{ title: t('lobby.titleRunning'), headerLargeTitle: false, headerBackVisible: false, gestureEnabled: false }} />
    </Stack>
  );
}
