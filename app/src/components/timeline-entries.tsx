import { Text } from 'react-native';

import type { TimelineEvent } from '@/api/types';
import { clockTime } from '@/lobby';
import { styles } from '@/styles';
import { timelineKey, timelineText } from '@/timeline';

/** One line per event, in the given order: clock time and what happened. */
export default function TimelineEntries({ events }: { events: TimelineEvent[] }) {
  return events.map((event) => (
    <Text key={timelineKey(event)} style={styles.text}>
      <Text style={styles.hint}>{clockTime(event.at)} · </Text>
      {timelineText(event)}
    </Text>
  ));
}
