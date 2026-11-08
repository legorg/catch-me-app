import { Stack } from 'expo-router';

// Registers the background location task before any screen renders.
import '@/background-location';

export default function RootLayout() {
  return <Stack screenOptions={{ headerShown: false }} />;
}
