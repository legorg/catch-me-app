import type { JokerSettings } from './api/types';

export type JokerKind = Exclude<keyof JokerSettings, 'zoneAlert'>;

/** Hunter team jokers, then runner jokers, in display order. */
export const HUNTER_JOKER_KINDS = ['showRunners', 'queryZone', 'compass', 'trap'] as const satisfies readonly JokerKind[];
export const RUNNER_JOKER_KINDS = ['zoneOnly', 'fakePosition', 'showHunters'] as const satisfies readonly JokerKind[];

/** Every allowance the backend accepts; 0 disables the joker. */
export const JOKER_ALLOWANCES = [0, 1, 2, 3, 4, 5] as const;

export const JOKER_TITLE_KEYS = {
  showRunners: 'joker.showRunners.title', queryZone: 'joker.queryZone.title', zoneOnly: 'joker.zoneOnly.title', fakePosition: 'joker.fakePosition.title',
  showHunters: 'joker.showHunters.title', compass: 'joker.compass.title', trap: 'joker.trap.title',
} as const satisfies Record<JokerKind, string>;

export const DEFAULT_JOKER_SETTINGS: JokerSettings = {
  showRunners: { allowance: 1 }, queryZone: { allowance: 1 }, compass: { allowance: 1 }, trap: { allowance: 1 },
  zoneOnly: { allowance: 1 }, fakePosition: { allowance: 1 }, showHunters: { allowance: 1 },
  zoneAlert: { enabled: true },
};

export function isValidJokerAllowance(allowance: unknown): allowance is number {
  return JOKER_ALLOWANCES.some((valid) => valid === allowance);
}

export function withAllowance(settings: JokerSettings, kind: JokerKind, allowance: number): JokerSettings {
  return { ...settings, [kind]: { ...settings[kind], allowance } };
}
