import { useEffect, useMemo, useState } from 'react';
import cronstrue from 'cronstrue';
import Modal from '../common/Modal';
import { useAuthContext } from '../../context/AuthContext';
import {
  useTestDetails,
  useTestNextRun,
  useUpdateTestSchedule,
} from '../../hooks/useApi';
import CronEditor from './CronEditor';
import type { TestSchedule } from '../../types/api';

// ScheduleCard is the shared display+editor for a test's schedule.
// It surfaces:
//   - whether a startup trigger is enabled
//   - whether the test runs off-queue when triggered
//   - each cron expression, with cronstrue's human-readable take
//   - the upcoming firing time (from the server's /next_run endpoint)
//
// When the user is logged in an "Edit" button opens the CronEditor
// in a modal. The save action PUTs to /api/v1/test/{id}/schedule.

interface ScheduleCardProps {
  testId: string;
  // Visual variant: 'card' renders inside a bordered card on the
  // library page; 'banner' is a flatter strip used on the runs page.
  variant?: 'card' | 'banner';
}

export function ScheduleCard({ testId, variant = 'card' }: ScheduleCardProps) {
  const { isLoggedIn } = useAuthContext();
  const detailsQuery = useTestDetails(testId, { enabled: !!testId });
  const nextRunQuery = useTestNextRun(testId, { enabled: !!testId });
  const update = useUpdateTestSchedule();

  const schedule = detailsQuery.data?.schedule ?? null;
  const [isEditing, setEditing] = useState(false);

  const earliestIn = useMemo(() => {
    if (!nextRunQuery.data?.earliest) return null;
    return relativeTime(nextRunQuery.data.earliest.next * 1000);
  }, [nextRunQuery.data]);

  const summary = summarizeSchedule(schedule);
  const isBanner = variant === 'banner';

  return (
    <>
      <div
        className={
          isBanner
            ? 'flex items-center justify-between gap-3 px-3 py-2 rounded bg-[var(--color-bg-secondary)] border border-[var(--color-border)]'
            : 'rounded border border-[var(--color-border)] bg-[var(--color-bg-secondary)] p-3 space-y-2'
        }
      >
        <div className={isBanner ? 'flex items-center gap-3 min-w-0' : ''}>
          <div className="flex items-center gap-2 text-xs uppercase tracking-wider font-semibold text-[var(--color-text-tertiary)] shrink-0">
            <ClockIcon className="size-3.5" /> Schedule
          </div>

          <div className={isBanner ? 'flex items-center gap-2 text-sm min-w-0' : 'text-sm'}>
            <span className="text-[var(--color-text-primary)] truncate">{summary}</span>
            {nextRunQuery.data?.earliest && (
              <span className="text-[var(--color-text-tertiary)] text-xs truncate">
                · next {earliestIn} ({new Date(nextRunQuery.data.earliest.next * 1000).toLocaleString()})
              </span>
            )}
          </div>
        </div>

        {!isBanner && schedule && schedule.cron && schedule.cron.length > 0 && (
          <ul className="text-xs space-y-0.5 pl-1">
            {schedule.cron.map((expr, idx) => (
              <li key={idx} className="text-[var(--color-text-secondary)]">
                <span className="font-mono text-[var(--color-text-primary)] mr-2">{expr}</span>
                <span className="text-[var(--color-text-tertiary)]">
                  {cronStringSafe(expr)}
                </span>
              </li>
            ))}
          </ul>
        )}

        {isLoggedIn && (
          <button
            type="button"
            onClick={() => setEditing(true)}
            className="btn btn-secondary btn-sm shrink-0"
          >
            Edit
          </button>
        )}
      </div>

      <ScheduleEditorModal
        isOpen={isEditing}
        testId={testId}
        initialSchedule={schedule}
        onClose={() => setEditing(false)}
        onSave={async (next) => {
          try {
            await update.mutateAsync({ testId, schedule: next });
            setEditing(false);
          } catch (err) {
            alert(`Failed to save schedule: ${err instanceof Error ? err.message : err}`);
          }
        }}
        isSaving={update.isPending}
      />
    </>
  );
}

// ScheduleEditorModal wraps CronEditor with a draft/cancel/save UX.
function ScheduleEditorModal({
  isOpen,
  testId,
  initialSchedule,
  onClose,
  onSave,
  isSaving,
}: {
  isOpen: boolean;
  testId: string;
  initialSchedule: TestSchedule | null;
  onClose: () => void;
  onSave: (s: TestSchedule | null) => void;
  isSaving: boolean;
}) {
  const [draft, setDraft] = useState<TestSchedule | null>(initialSchedule);

  useEffect(() => {
    setDraft(initialSchedule);
  }, [initialSchedule, isOpen]);

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`Schedule · ${testId}`} size="lg">
      <div className="space-y-4">
        <CronEditor schedule={draft} onChange={setDraft} />

        <div className="flex justify-between gap-2 pt-2 border-t border-[var(--color-border)]">
          <button
            type="button"
            onClick={() => setDraft(null)}
            className="btn btn-secondary btn-sm"
            disabled={isSaving}
          >
            Clear schedule
          </button>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={onClose}
              className="btn btn-secondary btn-sm"
              disabled={isSaving}
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={() => onSave(draft)}
              className="btn btn-primary btn-sm"
              disabled={isSaving}
            >
              {isSaving ? 'Saving…' : 'Save'}
            </button>
          </div>
        </div>
      </div>
    </Modal>
  );
}

// ── helpers ─────────────────────────────────────────────────────

function summarizeSchedule(schedule: TestSchedule | null): string {
  if (!schedule) return 'Not scheduled';
  const bits: string[] = [];
  if (schedule.startup) bits.push('runs on startup');
  if (schedule.cron && schedule.cron.length > 0) {
    bits.push(
      schedule.cron.length === 1
        ? '1 cron entry'
        : `${schedule.cron.length} cron entries`,
    );
  }
  if (schedule.skipQueue) bits.push('off-queue');
  return bits.length === 0 ? 'No triggers configured' : bits.join(' · ');
}

function cronStringSafe(expr: string): string {
  try {
    return cronstrue.toString(expr, { use24HourTimeFormat: true });
  } catch {
    return '(invalid expression)';
  }
}

// relativeTime returns a tiny string like "in 4m" / "in 2h" / "in 3d".
function relativeTime(unixMs: number): string {
  const diff = unixMs - Date.now();
  const abs = Math.abs(diff);
  const sign = diff < 0 ? 'ago' : 'in';
  if (abs < 60_000) return `${sign === 'in' ? 'in <1m' : 'just now'}`;
  if (abs < 3_600_000) return `${sign} ${Math.round(abs / 60_000)}m`;
  if (abs < 86_400_000) return `${sign} ${Math.round(abs / 3_600_000)}h`;
  return `${sign} ${Math.round(abs / 86_400_000)}d`;
}

function ClockIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <circle cx="12" cy="12" r="10" strokeWidth="2" />
      <path strokeWidth="2" strokeLinecap="round" d="M12 6v6l4 2" />
    </svg>
  );
}

export default ScheduleCard;
