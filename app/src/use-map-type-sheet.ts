import { useState } from 'react';

/** Open state of the map type sheet; choosing a type closes it. */
export function useMapTypeSheet<T>(onChange: (mapType: T) => void) {
  const [open, setOpen] = useState(false);
  const show = () => setOpen(true);
  const close = () => setOpen(false);
  const choose = (mapType: T) => {
    onChange(mapType);
    setOpen(false)
  };
  return { open, show, close, choose };
}
