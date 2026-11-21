import { useState } from 'react';
import { Image, View, type StyleProp, type ViewStyle } from 'react-native';

import pattern from '../../assets/camouflage.png';
import { camouflageStyles, dynamicStyles, showCamouflage } from '../styles';

/** Side length of assets/camouflage.png. */
const TILE = 320;

/**
 * Camouflage behind a screen's content, switched on in palette.json. The pattern only darkens
 * and lightens, so it takes its colour from the background beneath it.
 *
 * The tiles are laid out one by one: Android draws resizeMode="repeat" only once, and one
 * image as large as the area failed to draw on both platforms.
 */
export default function Camouflage({ style }: { style: StyleProp<ViewStyle> }) {
  const [size, setSize] = useState({ width: 0, height: 0 });
  if (!showCamouflage) return null;
  const columns = Math.ceil(size.width / TILE);
  const tiles = columns * Math.ceil(size.height / TILE);
  return (
    <View
      pointerEvents="none"
      style={[style, camouflageStyles.clip]}
      onLayout={({ nativeEvent }) => setSize({ width: nativeEvent.layout.width, height: nativeEvent.layout.height })}
    >
      <View style={[camouflageStyles.tiles, dynamicStyles.camouflageTiles(columns * TILE)]}>
        {Array.from({ length: tiles }, (_, index) => <Image key={index} source={pattern} style={camouflageStyles.tile} />)}
      </View>
    </View>
  );
}
