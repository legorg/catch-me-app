/** Does nothing; for props that must be given but have nothing to do. */
export const noop = () => {};

/** A handler that starts an async action and ignores its result (errors are the action's business). */
export const fireAndForget = (action: () => Promise<unknown>) => () => {
  void action();
};
