import { useState } from 'react';

import {
  ApiError, MAX_NAME_LENGTH, joinGame, setHunterTeam, setReady, updatePosition, type JoinCodes,
} from '@/api/client';
import { isGameStateRunnerResponse, type Bounds, type Coordinates, type Playground } from '@/api/types';
import { cellAt, rotatePoint } from '@/playground';

const MOVE_INTERVAL_MS = 5000;
// About 120 m per step, i.e. a brisk jog.
const STEP_DEGREES = 0.0011;
const MAX_TURN = 0.6;

type Bot = { token: string; position: Coordinates; playground: Playground; heading: number };

// Bots outlive the lobby, so they keep reporting after the game screen replaces it.
const bots = new Map<string, Bot>();
let mover: ReturnType<typeof setInterval> | undefined;

function randomIn({ north, south, east, west }: Bounds): Coordinates {
  return { latitude: south + Math.random() * (north - south), longitude: west + Math.random() * (east - west) };
}

function report(name: string, bot: Bot) {
  updatePosition(bot.token, { ...bot.position, accuracyMeters: 5 }).catch((err: unknown) => {
    // The game is gone (or the data was reset): stop moving this bot.
    if (err instanceof ApiError && err.status === 401) bots.delete(name);
  });
}

function addBot(name: string, bot: Bot) {
  bots.set(name, bot);
  report(name, bot);
  mover ??= setInterval(() => {
    for (const [botName, b] of bots) {
      // Keep going roughly the same way, turning a little each step.
      b.heading += (Math.random() - 0.5) * 2 * MAX_TURN;
      const next = {
        latitude: b.position.latitude + Math.sin(b.heading) * STEP_DEGREES,
        longitude: b.position.longitude + Math.cos(b.heading) * STEP_DEGREES,
      };
      // Turn around at the edge of the playground.
      if (cellAt(b.playground, next) === null) b.heading += Math.PI;
      else b.position = next;
      report(botName, b);
    }
    if (bots.size === 0) {
      clearInterval(mover);
      mover = undefined;
    }
  }, MOVE_INTERVAL_MS);
}

/** State and actions of the debug add-player dialog. */
export function useDebugAddPlayer(codes: JoinCodes) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState('Bot');
  const [team, setTeam] = useState<'runner' | 'hunter'>('hunter');
  const [hunterTeam, setHunterTeamName] = useState('Bots');
  const [ready, setReadyState] = useState(true);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState('');

  const add = async () => {
    setBusy(true);
    setMessage('');
    try {
      const { playerToken, state } = await joinGame(codes[team], name);
      if (team === 'hunter') await setHunterTeam(playerToken, hunterTeam);
      if (ready) await setReady(playerToken, true);
      const { playground } = state;
      // A random point of the unrotated grid, turned with it, always lies inside.
      const position = isGameStateRunnerResponse(state) ? state.route.start.coordinates : rotatePoint(playground, randomIn(playground.bounds));
      addBot(name.trim(), { token: playerToken, playground, position, heading: Math.random() * 2 * Math.PI });
      setMessage(`Added ${name.trim()}; it reports a position every 5 seconds.`);
      setName(nextName(name));
    } catch (err) {
      setMessage(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return {
    open, name, team, hunterTeam, ready, busy, message,
    setOpen, setName, setTeam, setHunterTeamName,
    show: () => setOpen(true),
    close: () => setOpen(false),
    toggleReady: () => setReadyState(!ready),
    add: () => void add(),
  };
}

/** "Bot" -> "Bot 2" -> "Bot 3", so names stay unique per game. */
function nextName(name: string): string {
  const match = /^(.*?)(?: (\d+))?$/.exec(name.trim());
  return `${match?.[1] ?? name} ${Number(match?.[2] ?? 1) + 1}`.slice(0, MAX_NAME_LENGTH);
}
