import Slider from '@react-native-community/slider';
import { Text, View } from 'react-native';

import { useT } from '@/i18n';
import { colors, styles } from '@/styles';

/** Card with a slider that turns the grid clockwise in 5° steps; the map above follows while dragging. */
export default function RotationCard({ degrees, onChange, disabled }: {
  degrees: number;
  onChange: (degrees: number) => void;
  disabled: boolean;
}) {
  const t = useT();
  return (
    <View style={styles.card}>
      <Text style={styles.heading}>{t('create.rotation')}</Text>
      <Text style={styles.text}>{t('create.rotationValue', { degrees })}</Text>
      <Slider
        accessibilityLabel={t('create.rotationLabel')}
        minimumValue={0}
        maximumValue={355}
        step={5}
        value={degrees}
        onValueChange={(value) => onChange(Math.round(value))}
        disabled={disabled}
        minimumTrackTintColor={colors.accent}
        maximumTrackTintColor={colors.placeholder}
        thumbTintColor={colors.accent}
      />
      <Text style={styles.hint}>{t('create.rotationHint')}</Text>
    </View>
  );
}
