import type { ReactNode } from 'react';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import FullscreenMap, { type FullscreenMapProps } from './fullscreen-map';
import KeyboardScrollView from './keyboard-scroll-view';

type Props = {
  /** The map at the top: square, from edge to edge, with its fullscreen button. */
  map: FullscreenMapProps;
  /** The screen's options below the map. */
  children: ReactNode;
};

/** A screen led by its map: the square map and the options below scroll together. */
export default function MapScrollView({ map, children }: Props) {
  const insets = useSafeAreaInsets();
  return (
    <KeyboardScrollView top={<FullscreenMap controlsInset={{ right: insets.right }} {...map} square />}>
      {children}
    </KeyboardScrollView>
  );
}
