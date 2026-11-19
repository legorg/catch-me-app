import { useRef, useState } from 'react';
import type { AccessibilityActionEvent, GestureResponderEvent, LayoutChangeEvent } from 'react-native';

/** Accessibility increment/decrement step, in the scrubber's unit. */
const STEP_SHARE = 0.05;

/**
 * Touch handling of a horizontal scrubber from 0 to `max`: tapping or dragging anywhere on it seeks there.
 * Uses page coordinates, so dragging beyond either end stays at that end.
 */
export function useScrubber(value: number, max: number, onSeek: (value: number) => void) {
  const [width, setWidth] = useState(0);
  const left = useRef(0);
  const seekTo = (pageX: number) => {
    if (width > 0) onSeek(Math.min(Math.max((pageX - left.current) / width, 0), 1) * max);
  };
  return {
    onLayout: (event: LayoutChangeEvent) => setWidth(event.nativeEvent.layout.width),
    onStartShouldSetResponder: () => true,
    onMoveShouldSetResponder: () => true,
    // A surrounding scroll view must not take over a drag.
    onResponderTerminationRequest: () => false,
    onResponderGrant: (event: GestureResponderEvent) => {
      left.current = event.nativeEvent.pageX - event.nativeEvent.locationX;
      seekTo(event.nativeEvent.pageX);
    },
    onResponderMove: (event: GestureResponderEvent) => seekTo(event.nativeEvent.pageX),
    accessibilityActions: [{ name: 'increment' as const }, { name: 'decrement' as const }],
    onAccessibilityAction: (event: AccessibilityActionEvent) =>
      onSeek(value + (event.nativeEvent.actionName === 'increment' ? 1 : -1) * STEP_SHARE * max),
  };
}
