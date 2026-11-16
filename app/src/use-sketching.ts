import { useCallback, useState } from 'react';

import { ApiError, MAX_SKETCH_POINTS, addSketch, clearSketches, type Coordinates, type GameStateResponse } from '@/api/client';
import { confirm } from '@/confirm';
import { useT } from '@/i18n';
import { thinPoints } from '@/lobby';

type Options = {
  token: string | null | undefined;
  show: (state: GameStateResponse) => void;
  fail: (err: unknown) => void;
};

/** Sketching (hunters in a hunter team): errors stay until the next action, unlike poll errors. */
export function useSketching({ token, show, fail }: Options) {
  const t = useT();
  const [drawing, setDrawing] = useState(false);
  const [error, setError] = useState('');
  const clearError = useCallback(() => setError(''), []);

  const failed = useCallback((err: unknown) => {
    if (err instanceof ApiError && err.status === 401) fail(err);
    else setError(err instanceof Error ? err.message : t('common.backendUnreachable'));
  }, [fail, t]);

  const send = useCallback(async (points: Coordinates[]) => {
    if (!token) return;
    setError('');
    try {
      show(await addSketch(token, thinPoints(points, MAX_SKETCH_POINTS)));
    } catch (err) {
      failed(err);
    }
  }, [token, show, failed]);

  const confirmClear = () => confirm(t('sketch.clearTitle'), t('sketch.clearText'), t('sketch.clearYes'), async () => {
    if (!token) return;
    setError('');
    try {
      show(await clearSketches(token));
    } catch (err) {
      failed(err);
    }
  });

  return { drawing, setDrawing, error, send, confirmClear, clearError };
}
