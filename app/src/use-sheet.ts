import { useEffect, useMemo, useState } from 'react';
import { Animated, useWindowDimensions } from 'react-native';

/** Fades the backdrop and slides the card in while open, and out once open turns false. */
export function useSheet(open: boolean) {
  const { height } = useWindowDimensions();
  // Stays true while the sheet animates out, so the modal closes only after the animation.
  const [visible, setVisible] = useState(open);
  if (open && !visible) setVisible(true);
  // 0 is hidden, 1 is fully shown.
  const [progress] = useState(() => new Animated.Value(0));
  const offset = useMemo(() => progress.interpolate({ inputRange: [0, 1], outputRange: [height, 0] }), [progress, height]);

  useEffect(() => {
    Animated.timing(progress, { toValue: open ? 1 : 0, duration: open ? 250 : 200, useNativeDriver: true }).start(() => {
      if (!open) setVisible(false);
    });
  }, [open, progress]);

  return { visible, backdropOpacity: progress, offset };
}
