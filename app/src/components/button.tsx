import { Pressable, Text } from 'react-native';

import { inform } from '../confirm';
import { styles } from '../styles';

/** Pressable style: the variant's look, dimmed while pressed or disabled. */
const buttonStyle = (primary: boolean, disabled: boolean) => ({ pressed }: { pressed: boolean }) =>
  [primary ? styles.button : styles.secondaryButton, (pressed || disabled) && styles.dimmed];

type ButtonProps = {
  label: string;
  onPress: () => void;
  disabled?: boolean;
  /** Why the button is disabled: pressing it then explains this in a system alert instead of doing nothing. */
  disabledReason?: string | null;
  variant?: 'primary' | 'secondary';
};

export default function Button({ label, onPress, disabled = false, disabledReason, variant = 'primary' }: ButtonProps) {
  const primary = variant === 'primary';
  const explains = disabled && !!disabledReason;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ disabled }}
      accessibilityHint={explains ? disabledReason : undefined}
      disabled={disabled && !explains}
      onPress={explains ? () => inform(label, disabledReason) : onPress}
      style={buttonStyle(primary, disabled)}
    >
      <Text style={primary ? styles.buttonText : styles.secondaryButtonText}>{label}</Text>
    </Pressable>
  );
}
