import { ScrollView, Text } from 'react-native';

import type { TimelineEvent } from '@/api/types';
import { useT } from '@/i18n';
import { clockTime } from '@/lobby';
import { overlay, styles } from '@/styles';
import { timelineKey, timelineText } from '@/timeline';

type TimelinePanelProps = {
  /** Newest first. */
  events: TimelineEvent[];
  loaded: boolean;
};

/** Collapsible card over the map with the game's events, for spectators. */
export default function TimelinePanel({ events, loaded }: TimelinePanelProps) {
  const t = useT();
  return (
    <ScrollView style={overlay.info} contentContainerStyle={styles.gapSmall}>
      <Text style={styles.heading}>{t('timeline.title')}</Text>
      {!loaded ? <Text style={styles.hint}>{t('timeline.loading')}</Text>
        : events.length === 0 ? <Text style={styles.hint}>{t('timeline.empty')}</Text>
        : events.map((event) => (
          <Text key={timelineKey(event)} style={styles.text}>
            <Text style={styles.hint}>{clockTime(event.at)} · </Text>
            {timelineText(event)}
          </Text>
        ))}
    </ScrollView>
  );
}
