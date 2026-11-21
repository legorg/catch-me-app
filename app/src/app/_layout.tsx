import { getLocales } from 'expo-localization';
import { Stack } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { Platform, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

// Registers the background location task before any screen renders.
import '@/background-location';
import LanguageSwitch from '@/components/language-switch';
import { isLanguage, setLanguage, useT } from '@/i18n';
import { colors, dynamicStyles, header } from '@/styles';

// Start in the first device language the app supports, otherwise English.
const deviceLanguage = getLocales().map((locale) => locale.languageCode).find(isLanguage);
if (deviceLanguage) setLanguage(deviceLanguage);

/** Android's header background: the native header cannot draw a bottom line itself. */
function HeaderBackground() {
  return <View style={header.background} />;
}

export default function RootLayout() {
  const t = useT();
  const insets = useSafeAreaInsets();
  return (
    <>
      {/* Light text on the dark screens. Not the Stack's statusBarStyle: on iOS that needs
          UIViewControllerBasedStatusBarAppearance, which Expo (and Expo Go) leave off. */}
      <StatusBar style="light" />
      <Stack
        screenOptions={{
          headerBlurEffect: 'dark',
          // The blur is iOS only; Android gets a dark bar with a khaki line instead.
          headerBackground: Platform.OS === 'android' ? HeaderBackground : undefined,
          headerTintColor: colors.white,
          headerBackButtonDisplayMode: 'minimal',
          headerTitleStyle: { fontWeight: '700' },
          headerShadowVisible: false,
          headerLargeTitle: true,
          headerLargeTitleShadowVisible: false,
          headerLargeTitleStyle: { color: colors.white },
          contentStyle: { backgroundColor: colors.background },
        }}
      >
        <Stack.Screen name="index" options={{ title: 'Catch me!', headerRight: () => <LanguageSwitch /> }} />
        <Stack.Screen name="create" options={{ title: t('create.chooseTitle') }} />
        {/* Joining replaces the previous screen, so there is nothing to go back to. */}
        <Stack.Screen name="lobby" options={{ title: t('lobby.title'), headerBackVisible: false, gestureEnabled: false }} />
        {/* The map fills the screen under its own overlay, so no header; leaving goes through the exit button. */}
        <Stack.Screen name="game" options={{ headerShown: false, gestureEnabled: false }} />
        <Stack.Screen name="results/[sessionId]" options={{ title: t('result.title'), headerBackVisible: false, gestureEnabled: false }} />
        {/* A plain modal over the game screen: one fixed height, swipe down to dismiss on iOS.
            Not a form sheet with several heights: on iOS its content went blank while dragging or scrolling. */}
        <Stack.Screen name="jokers" options={{
          presentation: 'modal', headerShown: false,
          contentStyle: { backgroundColor: colors.background },
        }} />
      </Stack>
      {/* Android draws edge-to-edge and ignores a navigation bar colour, so a solid strip sits behind its buttons. */}
      {Platform.OS === 'android' && <View pointerEvents="none" style={dynamicStyles.navigationBar(insets.bottom)} />}
    </>
  );
}
