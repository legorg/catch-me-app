import { ScrollView, Text } from 'react-native';

import type { TimelineEvent } from '@/api/types';
import TimelineEntries from '@/components/timeline-entries';
import { useT } from '@/i18n';
import { overlay, styles } from '@/styles';

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
        : <TimelineEntries events={events} />}
    </ScrollView>
  );
}
