import { Ionicons } from '@expo/vector-icons';
import { Pressable, type AccessibilityState } from 'react-native';

import { colors, overlay, styles } from '@/styles';

type HudIconButtonProps = {
  icon: keyof typeof Ionicons.glyphMap;
  size: number;
  label: string;
  onPress: () => void;
  accessibilityState?: AccessibilityState;
  disabled?: boolean;
};

/** Square icon button floating over the map. */
export default function HudIconButton({ icon, size, label, onPress, accessibilityState, disabled = false }: HudIconButtonProps) {
  return (
    <Pressable accessibilityRole="button" accessibilityLabel={label} accessibilityState={{ ...accessibilityState, disabled }} disabled={disabled} onPress={onPress} style={[overlay.icon, disabled && styles.dimmed]}>
      <Ionicons name={icon} size={size} color={colors.accent} />
    </Pressable>
  );
}
