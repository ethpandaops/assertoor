import { useEffect, useMemo, useState } from 'react';
import cronstrue from 'cronstrue';
import { CronExpressionParser } from 'cron-parser';
import type { TestSchedule } from '../../types/api';

// CronEditor edits a single TestSchedule — a Startup checkbox, an
// optional skip-queue toggle, and zero or more cron expressions.
// Each expression is validated and previewed (human-readable
// description + next 3 firings) inline.
//
// The user can pick from a few presets that fill in common
// expressions, or hand-write any standard 5-field crontab line.
//
// State is local to the editor; callers commit via onSave (typically
// after clicking the "Save" button in a containing modal).
interface CronEditorProps {
  schedule: TestSchedule | null;
  onChange: (next: TestSchedule | null) => void;
  // When true the editor is read-only; controls collapse to display.
  readOnly?: boolean;
}

// Preset crons that fill the input on click. We keep this short so the
// list stays scannable — anything beyond covers ~95% of use cases.
const PRESETS: { label: string; expr: string }[] = [
  { label: 'Every minute', expr: '* * * * *' },
  { label: 'Every 5 minutes', expr: '*/5 * * * *' },
  { label: 'Every 15 minutes', expr: '*/15 * * * *' },
  { label: 'Hourly', expr: '0 * * * *' },
  { label: 'Every 6 hours', expr: '0 */6 * * *' },
  { label: 'Daily at midnight', expr: '0 0 * * *' },
  { label: 'Weekly (Sun 00:00)', expr: '0 0 * * 0' },
];

export function CronEditor({ schedule, onChange, readOnly }: CronEditorProps) {
  const startup = schedule?.startup ?? false;
  const skipQueue = schedule?.skipQueue ?? false;
  const crons = schedule?.cron ?? [];

  // Mutators emit a fresh schedule object. We never mutate the
  // incoming one so the caller can rely on reference equality for
  // change detection.
  const patch = (changes: Partial<TestSchedule>) => {
    const next: TestSchedule = {
      startup,
      cron: crons,
      skipQueue,
      ...changes,
    };
    // If the schedule is empty (no startup, no crons, no skipQueue)
    // emit null so callers can clear the schedule entirely.
    if (!next.startup && !next.skipQueue && (!next.cron || next.cron.length === 0)) {
      onChange(null);
    } else {
      onChange(next);
    }
  };

  const setCronAt = (idx: number, expr: string) => {
    const next = [...crons];
    next[idx] = expr;
    patch({ cron: next });
  };

  const addCron = (expr: string) => patch({ cron: [...crons, expr] });
  const removeCron = (idx: number) => patch({ cron: crons.filter((_, i) => i !== idx) });

  return (
    <div className="space-y-3">
      {/* Toggles */}
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={startup}
            disabled={readOnly}
            onChange={(e) => patch({ startup: e.target.checked })}
          />
          Run on assertoor startup
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={skipQueue}
            disabled={readOnly}
            onChange={(e) => patch({ skipQueue: e.target.checked })}
          />
          Skip the runner queue (run in parallel)
        </label>
      </div>

      {/* Cron list */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium">Cron schedules</span>
          {!readOnly && (
            <PresetMenu onPick={(expr) => addCron(expr)} />
          )}
        </div>

        {crons.length === 0 && (
          <p className="text-xs text-[var(--color-text-tertiary)] italic">
            No cron schedule. Use a preset above or click &quot;Add expression&quot; to write your own.
          </p>
        )}

        {crons.map((expr, idx) => (
          <CronRow
            key={idx}
            value={expr}
            onChange={(v) => setCronAt(idx, v)}
            onRemove={() => removeCron(idx)}
            readOnly={readOnly}
          />
        ))}

        {!readOnly && (
          <button
            type="button"
            onClick={() => addCron('0 * * * *')}
            className="text-xs text-primary-600 hover:underline"
          >
            + Add expression
          </button>
        )}
      </div>
    </div>
  );
}

// CronRow handles a single cron expression: input + parsed description
// + next 3 firings. Errors are shown inline so the user knows when to
// stop saving.
function CronRow({
  value,
  onChange,
  onRemove,
  readOnly,
}: {
  value: string;
  onChange: (v: string) => void;
  onRemove: () => void;
  readOnly?: boolean;
}) {
  const { description, nextRuns, error } = useCronPreview(value);

  return (
    <div className="rounded border border-[var(--color-border)] bg-[var(--color-bg-secondary)] p-2 space-y-1">
      <div className="flex items-center gap-2">
        <input
          type="text"
          value={value}
          disabled={readOnly}
          onChange={(e) => onChange(e.target.value)}
          spellCheck={false}
          className={`flex-1 px-2 py-1 bg-[var(--color-bg-primary)] border rounded-sm text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary-500 ${
            error ? 'border-error-500' : 'border-[var(--color-border)]'
          }`}
          placeholder="* * * * *"
        />
        {!readOnly && (
          <button
            type="button"
            onClick={onRemove}
            className="px-2 py-1 text-xs text-error-600 hover:underline"
            title="Remove this expression"
          >
            ✕
          </button>
        )}
      </div>

      {error ? (
        <p className="text-xs text-error-600">{error}</p>
      ) : (
        <>
          <p className="text-xs text-[var(--color-text-secondary)]">
            {description}
          </p>
          {nextRuns.length > 0 && (
            <p className="text-[11px] text-[var(--color-text-tertiary)] font-mono">
              Next: {nextRuns.map((d) => d.toLocaleString()).join(' · ')}
            </p>
          )}
        </>
      )}
    </div>
  );
}

// PresetMenu is a tiny dropdown of common cron expressions. We render
// it as a `<details>` so we don't need any portal / outside-click
// handling. Picking a preset closes the menu and emits the chosen
// expression up to the parent.
function PresetMenu({ onPick }: { onPick: (expr: string) => void }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="btn btn-secondary btn-sm text-xs"
      >
        Add preset ▾
      </button>
      {open && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setOpen(false)} />
          <div className="absolute right-0 z-20 mt-1 w-56 rounded border border-[var(--color-border)] bg-[var(--color-bg-primary)] shadow-lg overflow-hidden">
            {PRESETS.map((p) => (
              <button
                key={p.expr}
                type="button"
                onClick={() => {
                  onPick(p.expr);
                  setOpen(false);
                }}
                className="w-full text-left px-3 py-1.5 text-xs hover:bg-[var(--color-bg-tertiary)]"
              >
                <span className="font-medium">{p.label}</span>
                <span className="ml-2 font-mono text-[var(--color-text-tertiary)]">
                  {p.expr}
                </span>
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
}

// useCronPreview parses + describes the expression on every change.
// Both cronstrue (description) and cron-parser (firing times) need to
// be tolerant of partial input — empty / mid-typing strings throw, so
// we surface those as a single 'error' string rather than rendering
// half-broken output.
function useCronPreview(value: string): {
  description: string;
  nextRuns: Date[];
  error: string | null;
} {
  const [tick, setTick] = useState(0);
  useEffect(() => {
    const t = setInterval(() => setTick((v) => v + 1), 30_000);
    return () => clearInterval(t);
  }, []);

  return useMemo(() => {
    void tick; // re-evaluate every 30s so next firings stay current
    const trimmed = value.trim();
    if (!trimmed) {
      return { description: '', nextRuns: [], error: 'empty expression' };
    }
    try {
      const description = cronstrue.toString(trimmed, {
        throwExceptionOnParseError: true,
        verbose: false,
        use24HourTimeFormat: true,
      });
      const parser = CronExpressionParser.parse(trimmed, { currentDate: new Date() });
      const nextRuns: Date[] = [];
      for (let i = 0; i < 3; i++) {
        nextRuns.push(parser.next().toDate());
      }
      return { description, nextRuns, error: null };
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      return { description: '', nextRuns: [], error: msg };
    }
  }, [value, tick]);
}

export default CronEditor;
