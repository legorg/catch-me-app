import { Text, View } from 'react-native';

import { dynamicStyles, styles as shared, sliderStyles as styles } from '../styles';
import { useSlider } from '../use-slider';

type Props = {
  title: string;
  /** Read out instead of the title, e.g. with the unit spelled out. */
  label: string;
  /** The value as shown next to the title. */
  valueText: string;
  value: number;
  min: number;
  max: number;
  step: number;
  onChange: (value: number) => void;
  disabled?: boolean;
};

/** A titled horizontal slider; the value snaps to whole steps. */
export default function Slider({ title, label, valueText, value, min, max, step, onChange, disabled = false }: Props) {
  const handlers = useSlider(value, min, max, step, onChange);
  const share = max > min ? (value - min) / (max - min) : 0;
  return (
    // Labelled like the other form fields (e.g. the joker allowances): plain text above the control.
    <View style={[shared.gapTiny, disabled && shared.dimmed]} pointerEvents={disabled ? 'none' : 'auto'}>
      <View style={styles.header}>
        <Text style={shared.text}>{title}</Text>
        <Text style={[shared.text, shared.accentText]}>{valueText}</Text>
      </View>
      <View
        style={styles.touch}
        accessible
        accessibilityRole="adjustable"
        accessibilityLabel={label}
        accessibilityValue={{ min, max, now: value, text: valueText }}
        {...handlers}
      >
        <View pointerEvents="none" style={styles.rail}>
          <View style={[styles.fill, dynamicStyles.sliderShare(share)]} />
        </View>
        <View pointerEvents="none" style={[styles.thumb, dynamicStyles.sliderThumb(share)]} />
      </View>
    </View>
  );
}
