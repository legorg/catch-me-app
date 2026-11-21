import { Text } from 'react-native';

import { overlay, styles } from '@/styles';

type ErrorPillProps = { message: string; live?: boolean };

function ErrorPill({ message, live = false }: ErrorPillProps) {
  if (!message) return null;
  return (
    <Text accessibilityLiveRegion={live ? 'polite' : undefined} style={[styles.text, styles.error, overlay.pill]}>{message}</Text>
  );
}

type ErrorPillsProps = { location: string; poll: string; sketch: string; timeline?: string };

/** Location, polling, sketch and timeline errors; empty ones are left out. */
export default function ErrorPills({ location, poll, sketch, timeline = '' }: ErrorPillsProps) {
  return (
    <>
      <ErrorPill message={location} />
      <ErrorPill message={poll} live />
      <ErrorPill message={sketch} live />
      <ErrorPill message={timeline} live />
    </>
  );
}
