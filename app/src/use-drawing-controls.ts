import { useState } from 'react';

import type { useSketching } from '@/use-sketching';

/** Info panel and drawing mode of the game screen: starting to draw closes the panel. */
export function useDrawingControls(sketching: ReturnType<typeof useSketching>) {
  const [infoOpen, setInfoOpen] = useState(false);

  const toggleInfo = () => setInfoOpen(!infoOpen);
  const startDrawing = () => {
    sketching.setDrawing(true);
    setInfoOpen(false);
    sketching.clearError();
  };
  const stopDrawing = () => {
    sketching.setDrawing(false);
    sketching.clearError();
  };

  return { infoOpen, toggleInfo, startDrawing, stopDrawing };
}
