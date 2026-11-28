import { Ionicons } from '@expo/vector-icons';
import { Pressable, Text, View } from 'react-native';

import Sheet from './sheet';
import { useT, type MessageKey } from '../i18n';
import { colors, mapStyles, picker } from '../styles';
import { useMapTypeSheet } from '../use-map-type-sheet';

export type MapType = 'standard' | 'satellite' | 'hybrid';

const OPTIONS: { type: MapType; icon: keyof typeof Ionicons.glyphMap; label: MessageKey }[] = [
  { type: 'standard', icon: 'map-outline', label: 'map.typeStandard' },
  { type: 'satellite', icon: 'earth-outline', label: 'map.typeSatellite' },
  { type: 'hybrid', icon: 'layers-outline', label: 'map.typeHybrid' },
];

type Props = {
  mapType: MapType;
  onChange: (mapType: MapType) => void;
};

/** A map control that opens a sheet to switch between the map, satellite and hybrid view. */
export default function MapTypeSheet({ mapType, onChange }: Props) {
  const t = useT();
  const { open, show, close, choose } = useMapTypeSheet(onChange);
  return (
    <>
      <Pressable accessibilityRole="button" accessibilityLabel={t('map.type')} onPress={show} style={mapStyles.locateButton}>
        <Ionicons name="layers" size={20} color={colors.background} />
      </Pressable>
      <Sheet open={open} onClose={close} title={t('map.type')}>
        <View style={picker.row}>
          {OPTIONS.map(({ type, icon, label }) => {
            const selected = type === mapType;
            return (
              <Pressable
                key={type}
                accessibilityRole="radio"
                accessibilityState={{ selected }}
                onPress={() => choose(type)}
                style={[picker.cell, picker.cellWide, mapStyles.typeOption, selected && picker.selected]}
              >
                <Ionicons name={icon} size={24} color={selected ? colors.background : colors.accent} />
                <Text style={[picker.label, selected && picker.selectedLabel]}>{t(label)}</Text>
              </Pressable>
            );
          })}
        </View>
      </Sheet>
    </>
  );
}
