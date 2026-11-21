import { Image, Platform, View, useWindowDimensions, type ImageStyle, type StyleProp } from 'react-native';

import pattern from '../../assets/camouflage.png';
import { camouflageStyles, dynamicStyles, showCamouflage } from '../styles';

/** Side length of assets/camouflage.png. */
const TILE = 320;

/**
 * Camouflage behind a screen's content, switched on in palette.json. The pattern only darkens
 * and lightens, so it takes its colour from the background beneath it.
 */
export default function Camouflage({ style }: { style: StyleProp<ImageStyle> }) {
  const { width, height } = useWindowDimensions();
  if (!showCamouflage) return null;
  if (Platform.OS === 'ios') {
    return <Image source={pattern} resizeMode="repeat" pointerEvents="none" style={style} />;
  }
  // Android draws resizeMode="repeat" only once, so the tiles are laid out one by one.
  const columns = Math.ceil(width / TILE);
  const tiles = columns * Math.ceil(height / TILE);
  return (
    <View pointerEvents="none" style={[style, camouflageStyles.clip]}>
      <View style={[camouflageStyles.tiles, dynamicStyles.camouflageTiles(columns * TILE)]}>
        {Array.from({ length: tiles }, (_, index) => <Image key={index} source={pattern} style={camouflageStyles.tile} />)}
      </View>
    </View>
  );
}
