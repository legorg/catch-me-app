import type { Stack } from 'expo-router';
import type { ComponentProps } from 'react';
import { Platform, View } from 'react-native';

import { colors, header } from './styles';

/** Android's header background: the native header cannot draw a bottom line itself. */
function HeaderBackground() {
  return <View style={header.background} />;
}

/** The header look of every stack, so the nested create stack matches the root one. */
export const stackScreenOptions: ComponentProps<typeof Stack>['screenOptions'] = {
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
};
