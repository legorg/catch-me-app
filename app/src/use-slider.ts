import { useRef, useState } from 'react';
import type { AccessibilityActionEvent, GestureResponderEvent, LayoutChangeEvent } from 'react-native';

/**
 * Touch handling of a horizontal slider from `min` to `max` in steps of `step`: tapping or dragging anywhere
 * on it moves the value there. Uses page coordinates, so dragging beyond either end stays at that end.
 */
export function useSlider(value: number, min: number, max: number, step: number, onChange: (value: number) => void) {
  const [width, setWidth] = useState(0);
  const left = useRef(0);
  const snap = (raw: number) => Math.min(Math.max(min + Math.round((raw - min) / step) * step, min), max);
  const change = (raw: number) => {
    const next = snap(raw);
    if (next !== value) onChange(next);
  };
  const seekTo = (pageX: number) => {
    if (width > 0) change(min + ((pageX - left.current) / width) * (max - min));
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
      change(value + (event.nativeEvent.actionName === 'increment' ? step : -step)),
  };
}
