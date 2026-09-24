import { useEffect, useState } from "react";

function sameValues(values: Record<string, unknown>, baseline: Record<string, unknown>) {
  const keys = new Set([...Object.keys(values), ...Object.keys(baseline)]);
  return [...keys].every((key) =>
    Object.is(values[key], baseline[key]) ||
    JSON.stringify(values[key]) === JSON.stringify(baseline[key]),
  );
}

interface State<T> {
  values: T;
  baseline: T;
  revision: string;
}

export function useSettingsDraft<T extends Record<string, unknown>>(
  incoming: T | undefined,
  revision: string | undefined,
) {
  const [state, setState] = useState<State<T> | null>(null);

  useEffect(() => {
    if (!incoming || !revision) return;
    setState((current) => {
      if (current?.revision === revision || (current && !sameValues(current.values, current.baseline))) {
        return current;
      }
      return { values: incoming, baseline: incoming, revision };
    });
  }, [incoming, revision]);

  return {
    values: state?.values,
    baseline: state?.baseline,
    revision: state?.revision,
    dirty: state ? !sameValues(state.values, state.baseline) : false,
    change(values: T) {
      setState((current) => current ? { ...current, values } : current);
    },
    saved(values: T, revision: string) {
      setState((current) => ({ values: current?.values ?? values, baseline: values, revision }));
    },
    reset(values: T, revision: string) {
      setState({ values, baseline: values, revision });
    },
  };
}
