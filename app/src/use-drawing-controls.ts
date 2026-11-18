import { useState } from 'react';

import type { useSketching } from '@/use-sketching';

type Panel = 'info' | 'timeline';

/**
 * Info panel, timeline panel and drawing mode of the game screen. At most one
 * panel is open; starting to draw closes it.
 */
export function useDrawingControls(sketching: ReturnType<typeof useSketching>) {
  const [panel, setPanel] = useState<Panel | null>(null);

  const toggle = (next: Panel) => setPanel(panel === next ? null : next);
  const startDrawing = () => {
    sketching.setDrawing(true);
    setPanel(null);
    sketching.clearError();
  };
  const stopDrawing = () => {
    sketching.setDrawing(false);
    sketching.clearError();
  };

  return {
    infoOpen: panel === 'info', toggleInfo: () => toggle('info'),
    timelineOpen: panel === 'timeline', toggleTimeline: () => toggle('timeline'),
    startDrawing, stopDrawing,
  };
}
