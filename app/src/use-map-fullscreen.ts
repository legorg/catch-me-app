import { useState } from 'react';

/** Open state of the fullscreen map; closing also remounts the inline map via `inlineKey`. */
export function useMapFullscreen() {
  const [open, setOpen] = useState(false);
  // The grid may have moved in fullscreen; remounting the small map centers and fits it on the grid again.
  const [inlineKey, setInlineKey] = useState(0);
  const close = () => {
    setOpen(false);
    setInlineKey((key) => key + 1);
  };
  return { open, setOpen, inlineKey, close };
}
