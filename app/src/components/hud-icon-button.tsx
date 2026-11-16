import { Ionicons } from '@expo/vector-icons';
import { Pressable, type AccessibilityState } from 'react-native';

import { colors, overlay } from '@/styles';

type HudIconButtonProps = {
  icon: keyof typeof Ionicons.glyphMap;
  size: number;
  label: string;
  onPress: () => void;
  accessibilityState?: AccessibilityState;
};

/** Square icon button floating over the map. */
export default function HudIconButton({ icon, size, label, onPress, accessibilityState }: HudIconButtonProps) {
  return (
    <Pressable accessibilityRole="button" accessibilityLabel={label} accessibilityState={accessibilityState} onPress={onPress} style={overlay.icon}>
      <Ionicons name={icon} size={size} color={colors.accent} />
    </Pressable>
  );
}
