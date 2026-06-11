import { useEffect, useState } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import CodeMirror from '@uiw/react-codemirror';
import { yaml as yamlLang } from '@codemirror/lang-yaml';
import { EditorView } from '@codemirror/view';
import { EditorState } from '@codemirror/state';
import { githubLight, githubDark } from '@uiw/codemirror-theme-github';
import { useDarkMode } from '../../hooks/useDarkMode';
import Modal from '../common/Modal';
import type { LibraryEntry } from '../../types/api';
import type { TreeNode } from './libraryTreeUtils';
import { countPlaybooks } from './libraryTreeUtils';

export type LibrarySelection =
  | { kind: 'playbook'; playbook: LibraryEntry }
  | { kind: 'folder'; folder: TreeNode };

interface LibraryDetailsProps {
  selection: LibrarySelection | null;
  registered: boolean;
  canImport: boolean;
  baseURL: string;
  busy: 'import' | 'import-and-run' | null;
  onImport: (playbook: LibraryEntry) => void;
  onImportAndRun: (playbook: LibraryEntry) => void;
  onPlaybookSelect: (playbook: LibraryEntry) => void;
}

function LibraryDetails(props: LibraryDetailsProps) {
  if (!props.selection) {
    return (
      <div className="h-full flex items-center justify-center p-12 text-center text-[var(--color-text-secondary)]">
        <div>
          <p className="text-sm">Pick a playbook or folder from the tree to see details.</p>
          <p className="text-xs mt-2 text-[var(--color-text-tertiary)]">
            Use the search and tag filters at the top to narrow the list.
          </p>
        </div>
      </div>
    );
  }

  if (props.selection.kind === 'playbook') {
    return <PlaybookDetails {...props} playbook={props.selection.playbook} />;
  }

  return (
    <FolderDetails
      folder={props.selection.folder}
      onPlaybookSelect={props.onPlaybookSelect}
    />
  );
}

interface PlaybookDetailsProps extends LibraryDetailsProps {
  playbook: LibraryEntry;
}

function PlaybookDetails({
  playbook,
  registered,
  canImport,
  baseURL,
  busy,
  onImport,
  onImportAndRun,
}: PlaybookDetailsProps) {
  const isDev = (playbook.version ?? '').startsWith('0.');
  const remoteURL = baseURL + playbook.file;
  const [yamlOpen, setYamlOpen] = useState(false);
  const [copied, setCopied] = useState(false);

  const handleCopyLink = async () => {
    try {
      await navigator.clipboard.writeText(remoteURL);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      window.prompt('Copy the playbook URL', remoteURL);
    }
  };

  return (
    <div className="p-4 md:p-6 space-y-4">
      <div className="space-y-2">
        <div className="flex items-start justify-between gap-3">
          <h2 className="text-lg font-semibold leading-tight">{playbook.name}</h2>
          {playbook.version && (
            <span
              className={`px-2 py-0.5 rounded-xs text-xs font-medium flex-shrink-0 ${
                isDev
                  ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/40 dark:text-yellow-200'
                  : 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200'
              }`}
              title={isDev ? 'In development' : 'Production-ready'}
            >
              v{playbook.version}
            </span>
          )}
        </div>

        <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-[var(--color-text-secondary)]">
          <code className="font-mono">{playbook.file}</code>
          {playbook.timeout && <span>· timeout {playbook.timeout}</span>}
          {registered && (
            <span className="text-primary-600 font-medium">· already registered locally</span>
          )}
        </div>

        {playbook.tags && playbook.tags.length > 0 && (
          <div className="flex flex-wrap gap-1.5 pt-1">
            {playbook.tags.map((tag) => (
              <span
                key={tag}
                className="px-2 py-0.5 bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] rounded-xs text-xs"
              >
                {tag}
              </span>
            ))}
          </div>
        )}
      </div>

      <div className="border-t border-[var(--color-border)] pt-4">
        {playbook.description ? (
          <div className="markdown-body">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{playbook.description}</ReactMarkdown>
          </div>
        ) : (
          <p className="text-sm text-[var(--color-text-tertiary)] italic">No description provided.</p>
        )}
      </div>

      <div className="border-t border-[var(--color-border)] pt-4 flex flex-wrap items-center gap-2">
        <button
          onClick={() => setYamlOpen(true)}
          className="btn btn-secondary btn-sm flex items-center gap-1.5"
          title="View source YAML"
        >
          <CodeIcon className="size-4" />
          View YAML
        </button>
        <button
          onClick={handleCopyLink}
          className="btn btn-secondary btn-sm flex items-center gap-1.5"
          title="Copy playbook URL to clipboard"
        >
          <LinkIcon className="size-4" />
          {copied ? 'Copied!' : 'Copy Link'}
        </button>

        <div className="flex-1" />

        {canImport ? (
          <>
            <button
              onClick={() => onImport(playbook)}
              disabled={busy !== null}
              className="btn btn-secondary btn-sm"
            >
              {busy === 'import' ? 'Importing…' : 'Import'}
            </button>
            <button
              onClick={() => onImportAndRun(playbook)}
              disabled={busy !== null}
              className="btn btn-primary btn-sm flex items-center gap-1.5"
            >
              <PlayIcon className="size-4" />
              {busy === 'import-and-run' ? 'Importing…' : 'Import & Run'}
            </button>
          </>
        ) : (
          <span className="text-xs text-[var(--color-text-tertiary)]">
            Log in to import or run this playbook.
          </span>
        )}
      </div>

      <YamlViewerModal
        open={yamlOpen}
        onClose={() => setYamlOpen(false)}
        title={playbook.name}
        url={remoteURL}
      />
    </div>
  );
}

interface YamlViewerModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  url: string;
}

// YamlViewerModal fetches the remote playbook YAML on first open and
// renders it in a read-only CodeMirror editor with YAML syntax
// highlighting. The fetched text is reset when the modal closes so
// re-opening always reflects the current upstream.
function YamlViewerModal({ open, onClose, title, url }: YamlViewerModalProps) {
  const isDarkMode = useDarkMode();
  const cmTheme = isDarkMode ? githubDark : githubLight;
  const [content, setContent] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!open) return;

    let cancelled = false;
    setError(null);
    setContent(null);

    (async () => {
      try {
        const resp = await fetch(url);
        if (!resp.ok) {
          throw new Error(`HTTP ${resp.status} ${resp.statusText}`);
        }
        const text = await resp.text();
        if (!cancelled) setContent(text);
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err));
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [open, url]);

  const handleCopy = async () => {
    if (!content) return;
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // ignore
    }
  };

  return (
    <Modal isOpen={open} onClose={onClose} title={`Playbook: ${title}`} size="xl">
      <div className="space-y-3">
        <div className="flex items-center gap-2 text-xs text-[var(--color-text-secondary)]">
          <code className="font-mono break-all">{url}</code>
        </div>

        {error ? (
          <div className="p-3 bg-error-50 dark:bg-error-900/20 border border-error-200 dark:border-error-800 rounded-xs text-sm text-error-600">
            Failed to load YAML: {error}
          </div>
        ) : content === null ? (
          <div className="p-6 text-center text-sm text-[var(--color-text-secondary)]">
            Loading…
          </div>
        ) : (
          <div className="border border-[var(--color-border)] rounded-sm overflow-hidden">
            <CodeMirror
              value={content}
              height="60vh"
              extensions={[yamlLang(), EditorView.editable.of(false), EditorState.readOnly.of(true)]}
              theme={cmTheme}
              basicSetup={{
                lineNumbers: true,
                foldGutter: true,
                highlightActiveLine: false,
              }}
              className="text-sm"
            />
          </div>
        )}

        <div className="flex justify-end gap-2">
          {content !== null && (
            <button onClick={handleCopy} className="btn btn-secondary btn-sm">
              {copied ? 'Copied!' : 'Copy YAML'}
            </button>
          )}
          <button onClick={onClose} className="btn btn-primary btn-sm">
            Close
          </button>
        </div>
      </div>
    </Modal>
  );
}

interface FolderDetailsProps {
  folder: TreeNode;
  onPlaybookSelect: (playbook: LibraryEntry) => void;
}

function FolderDetails({ folder, onPlaybookSelect }: FolderDetailsProps) {
  const total = countPlaybooks(folder);
  const direct = folder.playbooks.length;
  const subfolders = folder.children.length;

  // Aggregate the tags used by playbooks (direct + descendants) so users
  // can quickly see what tags a folder spans.
  const tagCounts = new Map<string, number>();
  const visit = (node: TreeNode) => {
    for (const pb of node.playbooks) {
      for (const tag of pb.tags ?? []) {
        tagCounts.set(tag, (tagCounts.get(tag) ?? 0) + 1);
      }
    }
    for (const child of node.children) visit(child);
  };
  visit(folder);
  const topTags = Array.from(tagCounts.entries())
    .sort((a, b) => b[1] - a[1])
    .slice(0, 12);

  return (
    <div className="p-4 md:p-6 space-y-4">
      <div className="space-y-2">
        <h2 className="text-lg font-semibold leading-tight">{folder.name}</h2>
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-[var(--color-text-secondary)]">
          <code className="font-mono">{folder.path}/</code>
          <span>·</span>
          <span>
            {total} playbook{total === 1 ? '' : 's'}
          </span>
          {subfolders > 0 && (
            <>
              <span>·</span>
              <span>
                {subfolders} subfolder{subfolders === 1 ? '' : 's'}
              </span>
            </>
          )}
          {direct > 0 && direct !== total && (
            <>
              <span>·</span>
              <span>{direct} direct</span>
            </>
          )}
        </div>
      </div>

      <div className="border-t border-[var(--color-border)] pt-4">
        {folder.description ? (
          <div className="markdown-body">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{folder.description}</ReactMarkdown>
          </div>
        ) : (
          <p className="text-sm text-[var(--color-text-tertiary)] italic">No folder description.</p>
        )}
      </div>

      {topTags.length > 0 && (
        <div className="border-t border-[var(--color-border)] pt-4">
          <div className="text-xs font-medium text-[var(--color-text-secondary)] mb-2">
            Common tags
          </div>
          <div className="flex flex-wrap gap-1.5">
            {topTags.map(([tag, count]) => (
              <span
                key={tag}
                className="px-2 py-0.5 bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] rounded-xs text-xs"
              >
                {tag}
                <span className="ml-1 text-[var(--color-text-tertiary)]">{count}</span>
              </span>
            ))}
          </div>
        </div>
      )}

      {direct > 0 && (
        <div className="border-t border-[var(--color-border)] pt-4">
          <div className="text-xs font-medium text-[var(--color-text-secondary)] mb-2">
            Playbooks in this folder
          </div>
          <ul className="space-y-1">
            {folder.playbooks.map((pb) => {
              const isDev = (pb.version ?? '').startsWith('0.');
              return (
                <li key={pb.file}>
                  <button
                    type="button"
                    onClick={() => onPlaybookSelect(pb)}
                    className="w-full text-left px-2 py-1.5 hover:bg-[var(--color-bg-tertiary)] rounded-xs flex items-center gap-2"
                  >
                    <span
                      className={`size-2 rounded-full flex-shrink-0 ${
                        isDev ? 'bg-yellow-500' : 'bg-emerald-500'
                      }`}
                    />
                    <span className="text-sm flex-1 truncate">{pb.name}</span>
                    {pb.version && (
                      <span className="text-xs text-[var(--color-text-tertiary)]">v{pb.version}</span>
                    )}
                  </button>
                </li>
              );
            })}
          </ul>
        </div>
      )}
    </div>
  );
}

function CodeIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"
      />
    </svg>
  );
}

function LinkIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"
      />
    </svg>
  );
}

function PlayIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
      />
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
      />
    </svg>
  );
}

export default LibraryDetails;
