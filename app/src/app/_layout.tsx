import { getLocales } from 'expo-localization';
import { Stack } from 'expo-router';

// Registers the background location task before any screen renders.
import '@/background-location';
import { isLanguage, setLanguage } from '@/i18n';

// Start in the first device language the app supports, otherwise English.
const deviceLanguage = getLocales().map((locale) => locale.languageCode).find(isLanguage);
if (deviceLanguage) setLanguage(deviceLanguage);

export default function RootLayout() {
  return <Stack screenOptions={{ headerShown: false }} />;
}
