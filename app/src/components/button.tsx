import { Pressable, Text } from 'react-native';

import { styles } from '../styles';

type ButtonProps = {
  label: string;
  onPress: () => void;
  disabled?: boolean;
  variant?: 'primary' | 'secondary';
};

export default function Button({ label, onPress, disabled = false, variant = 'primary' }: ButtonProps) {
  const primary = variant === 'primary';
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ disabled }}
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => [primary ? styles.button : styles.secondaryButton, (pressed || disabled) && styles.dimmed]}
    >
      <Text style={primary ? styles.buttonText : styles.secondaryButtonText}>{label}</Text>
    </Pressable>
  );
}
