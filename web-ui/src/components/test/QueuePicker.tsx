import { useEffect, useMemo, useRef, useState } from 'react';
import { useTestQueue } from '../../hooks/useApi';
import type { ScheduleQueueOption, TestQueueEntry } from '../../types/api';

// QueuePicker is a 2-level nested dropdown bound to the live runner
// queue. The top-level options are:
//
//   - Run immediately            (creates a parallel off-queue slot)
//   - Add to queue → End of queue
//   - Add to queue → After <run #N name>     (one entry per live/queued test)
//
// We render this as a custom dropdown (not a native <select>) so the
// nested second level can show rich content (run id, name, status
// dot). Outside clicks close it.
interface QueuePickerProps {
  value: ScheduleQueueOption;
  onChange: (v: ScheduleQueueOption) => void;
}

export function QueuePicker({ value, onChange }: QueuePickerProps) {
  const { data, isLoading } = useTestQueue();
  const queue = useMemo(() => data?.queue ?? [], [data]);

  const [open, setOpen] = useState(false);
  const [activeGroup, setActiveGroup] = useState<'main' | 'after'>('main');
  const rootRef = useRef<HTMLDivElement>(null);

  // Close on outside click + ESC.
  useEffect(() => {
    if (!open) return;
    const onDocClick = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    const onEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false);
    };
    document.addEventListener('mousedown', onDocClick);
    document.addEventListener('keydown', onEsc);
    return () => {
      document.removeEventListener('mousedown', onDocClick);
      document.removeEventListener('keydown', onEsc);
    };
  }, [open]);

  // Lookup the run referenced by `after_run_id` so we can label the
  // collapsed picker correctly even when the queue list is stale or
  // long.
  const afterEntry = useMemo<TestQueueEntry | undefined>(() => {
    if (value.mode !== 'after' || !value.after_run_id) return undefined;
    return queue.find((q) => q.run_id === value.after_run_id);
  }, [queue, value]);

  const label = describeChoice(value, afterEntry);
  const hint = describeChoiceHint(value);

  return (
    <div className="relative" ref={rootRef}>
      <button
        type="button"
        onClick={() => {
          setOpen((v) => !v);
          setActiveGroup('main');
        }}
        className="w-full flex items-center justify-between gap-2 px-3 py-2 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-sm text-sm hover:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
      >
        <span className="flex flex-col items-start min-w-0">
          <span className="font-medium truncate">{label}</span>
          {hint && (
            <span className="text-[11px] text-[var(--color-text-tertiary)] truncate">
              {hint}
            </span>
          )}
        </span>
        <span className="text-[var(--color-text-tertiary)]">▾</span>
      </button>

      {open && (
        // The dropdown lives in-flow (not `absolute`) so it pushes any
        // following content down — that way the surrounding modal's
        // own scroll picks it up when the picker overflows the
        // viewport, rather than clipping silently.
        <div className="mt-1 rounded border border-[var(--color-border)] bg-[var(--color-bg-primary)] shadow-lg overflow-hidden">
          {activeGroup === 'main' ? (
            <div role="menu">
              <Option
                title="Run immediately"
                description="Parallel slot — bypasses the queue."
                selected={value.mode === 'immediate'}
                onClick={() => {
                  onChange({ mode: 'immediate' });
                  setOpen(false);
                }}
              />
              <Option
                title="Add to queue · End"
                description="Append behind every queued test."
                selected={value.mode === 'end'}
                onClick={() => {
                  onChange({ mode: 'end' });
                  setOpen(false);
                }}
              />
              <Option
                title="Add to queue · After…"
                description={
                  queue.length === 0
                    ? 'No queued or running tests yet.'
                    : `Insert behind a running or queued test (${queue.length} candidate${queue.length === 1 ? '' : 's'}).`
                }
                disabled={queue.length === 0}
                selected={value.mode === 'after'}
                trailing="›"
                onClick={() => {
                  if (queue.length === 0) return;
                  setActiveGroup('after');
                }}
              />
            </div>
          ) : (
            <div role="menu">
              <button
                type="button"
                onClick={() => setActiveGroup('main')}
                className="w-full text-left px-3 py-2 text-xs uppercase tracking-wider text-[var(--color-text-tertiary)] hover:bg-[var(--color-bg-tertiary)] border-b border-[var(--color-border)]"
              >
                ‹ Back · Insert after…
              </button>
              {isLoading ? (
                <p className="px-3 py-2 text-xs text-[var(--color-text-tertiary)]">Loading queue…</p>
              ) : (
                <div className="max-h-80 overflow-y-auto">
                  {queue.map((q) => (
                    <QueueOption
                      key={q.run_id}
                      entry={q}
                      selected={value.mode === 'after' && value.after_run_id === q.run_id}
                      onClick={() => {
                        onChange({ mode: 'after', after_run_id: q.run_id });
                        setOpen(false);
                      }}
                    />
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ── primitives ───────────────────────────────────────────────────

function Option({
  title,
  description,
  selected,
  disabled,
  trailing,
  onClick,
}: {
  title: string;
  description: string;
  selected?: boolean;
  disabled?: boolean;
  trailing?: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      role="menuitem"
      onClick={onClick}
      disabled={disabled}
      className={`w-full text-left px-3 py-1.5 hover:bg-[var(--color-bg-tertiary)] disabled:opacity-40 disabled:cursor-not-allowed border-b border-[var(--color-border)] last:border-b-0 ${
        selected ? 'bg-primary-50 dark:bg-primary-900/20' : ''
      }`}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="font-medium text-sm">{title}</span>
        <div className="flex items-center gap-2 shrink-0">
          {selected && !trailing && (
            <span className="text-primary-600 text-[10px] uppercase tracking-wider">selected</span>
          )}
          {trailing && <span className="text-[var(--color-text-tertiary)]">{trailing}</span>}
        </div>
      </div>
      <p className="text-[11px] text-[var(--color-text-tertiary)]">{description}</p>
    </button>
  );
}

function QueueOption({
  entry,
  selected,
  onClick,
}: {
  entry: TestQueueEntry;
  selected: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      role="menuitem"
      onClick={onClick}
      className={`w-full text-left px-3 py-2 hover:bg-[var(--color-bg-tertiary)] border-b border-[var(--color-border)] last:border-b-0 ${
        selected ? 'bg-primary-50 dark:bg-primary-900/20' : ''
      }`}
    >
      <div className="flex items-center gap-2 min-w-0">
        <StatusDot status={entry.status} />
        <span className="font-mono text-xs text-primary-600 shrink-0">#{entry.run_id}</span>
        <span className="text-sm truncate flex-1">{entry.name}</span>
        <span className="text-[10px] uppercase tracking-wider text-[var(--color-text-tertiary)] shrink-0">
          {entry.status}
        </span>
      </div>
    </button>
  );
}

function StatusDot({ status }: { status: string }) {
  const cls =
    status === 'running'
      ? 'bg-blue-500 animate-pulse'
      : status === 'pending'
        ? 'bg-gray-400'
        : status === 'success'
          ? 'bg-green-500'
          : 'bg-red-500';
  return <span className={`size-2 rounded-full shrink-0 ${cls}`} title={status} />;
}

// ── helpers ──────────────────────────────────────────────────────

function describeChoice(v: ScheduleQueueOption, after?: TestQueueEntry): string {
  switch (v.mode) {
    case 'immediate':
      return 'Run immediately';
    case 'end':
      return 'Add to queue · End of queue';
    case 'after':
      return after
        ? `Add to queue · After #${after.run_id} (${after.name})`
        : `Add to queue · After #${v.after_run_id ?? '?'}`;
  }
}

function describeChoiceHint(v: ScheduleQueueOption): string {
  switch (v.mode) {
    case 'immediate':
      return 'Parallel execution; bypasses the runner queue.';
    case 'end':
      return 'Sequential; runs after every currently queued test.';
    case 'after':
      return 'Sequential; inserted right after the selected test.';
  }
}

export default QueuePicker;
