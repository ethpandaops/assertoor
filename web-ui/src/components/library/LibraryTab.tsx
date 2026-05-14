import { useEffect, useMemo, useState, useCallback } from 'react';
import {
  usePlaybookLibrary,
  useTests,
  useCheckPlaybookLibrary,
  useRegisterExternalTest,
} from '../../hooks/useApi';
import { useAuthContext } from '../../context/AuthContext';
import SplitPane from '../common/SplitPane';
import Modal from '../common/Modal';
import StartTestModal from '../test/StartTestModal';
import type { LibraryEntry, LibraryCheckResponse } from '../../types/api';
import {
  buildTree,
  collectTags,
  filterTree,
  type LibraryFilter,
  type SearchScope,
  type TreeNode,
} from './libraryTree';
import LibraryTree from './LibraryTree';
import LibraryDetails, { type LibrarySelection } from './LibraryDetails';

const EXPANDED_KEY_PREFIX = 'library:expanded:';
const SCOPE_KEY = 'library:searchScope';

function loadInitialExpanded(): Set<string> {
  const set = new Set<string>();
  if (typeof window === 'undefined') return set;
  for (let i = 0; i < window.localStorage.length; i++) {
    const key = window.localStorage.key(i);
    if (key && key.startsWith(EXPANDED_KEY_PREFIX)) {
      const path = key.slice(EXPANDED_KEY_PREFIX.length);
      if (window.localStorage.getItem(key) === '1') {
        set.add(path);
      }
    }
  }
  return set;
}

function persistExpanded(path: string, expanded: boolean): void {
  if (typeof window === 'undefined') return;
  if (expanded) {
    window.localStorage.setItem(EXPANDED_KEY_PREFIX + path, '1');
  } else {
    window.localStorage.removeItem(EXPANDED_KEY_PREFIX + path);
  }
}

function loadInitialScope(): SearchScope {
  if (typeof window === 'undefined') return 'tags';
  const stored = window.localStorage.getItem(SCOPE_KEY);
  return stored === 'name' || stored === 'all' ? stored : 'tags';
}

// ancestorsWithMatches returns the set of folder paths that should be
// force-expanded so every matching playbook is visible. We mark every
// ancestor folder of any node containing at least one matching playbook
// or matching descendant.
function ancestorsWithMatches(root: TreeNode): Set<string> {
  const result = new Set<string>();
  const walk = (node: TreeNode, ancestors: string[]): boolean => {
    let hasMatch = node.playbooks.length > 0;
    for (const child of node.children) {
      if (walk(child, [...ancestors, node.path])) hasMatch = true;
    }
    if (hasMatch) {
      for (const a of ancestors) {
        if (a !== '') result.add(a);
      }
      if (node.path !== '') result.add(node.path);
    }
    return hasMatch;
  };
  walk(root, []);
  return result;
}

function LibraryTab() {
  const { isLoggedIn } = useAuthContext();
  const libraryQuery = usePlaybookLibrary();
  const testsQuery = useTests();
  const checkMutation = useCheckPlaybookLibrary();
  const registerMutation = useRegisterExternalTest();

  const [search, setSearch] = useState('');
  const [scope, setScope] = useState<SearchScope>(loadInitialScope);
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [selection, setSelection] = useState<LibrarySelection | null>(null);
  const [expanded, setExpanded] = useState<Set<string>>(loadInitialExpanded);

  // Import flow state
  const [runTestId, setRunTestId] = useState<string | null>(null);
  const [busy, setBusy] = useState<'import' | 'import-and-run' | null>(null);
  const [overwrite, setOverwrite] = useState<{
    check: LibraryCheckResponse;
    playbook: LibraryEntry;
    runAfter: boolean;
  } | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  useEffect(() => {
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(SCOPE_KEY, scope);
    }
  }, [scope]);

  // Mark library files whose id is already locally registered.
  const registeredFiles = useMemo(() => {
    const set = new Set<string>();
    if (!libraryQuery.data || !testsQuery.data) return set;
    const localIds = new Set(testsQuery.data.map((t) => t.id));
    for (const pb of libraryQuery.data.playbooks) {
      if (localIds.has(pb.id)) set.add(pb.file);
    }
    return set;
  }, [libraryQuery.data, testsQuery.data]);

  const filter: LibraryFilter = useMemo(
    () => ({ search, scope, tags: selectedTags }),
    [search, scope, selectedTags],
  );

  const tree = useMemo(() => {
    if (!libraryQuery.data) return null;
    return buildTree(libraryQuery.data.folders, libraryQuery.data.playbooks);
  }, [libraryQuery.data]);

  const filteredTree = useMemo(() => {
    if (!tree) return null;
    return filterTree(tree, filter) ?? { ...tree, playbooks: [], children: [] };
  }, [tree, filter]);

  const allTags = useMemo(() => {
    if (!libraryQuery.data) return [];
    return collectTags(libraryQuery.data.playbooks);
  }, [libraryQuery.data]);

  const filterIsActive = filter.search.trim() !== '' || filter.tags.length > 0;
  const forceExpand = useMemo(() => {
    if (!filterIsActive || !filteredTree) return new Set<string>();
    return ancestorsWithMatches(filteredTree);
  }, [filterIsActive, filteredTree]);

  // Selecting a folder both highlights it in the details pane AND
  // toggles its expand state in a single click. Forced-open folders
  // (due to active filters) can't be collapsed.
  const handleSelectFolder = useCallback(
    (folder: TreeNode) => {
      setSelection({ kind: 'folder', folder });
      if (forceExpand.has(folder.path)) return;
      setExpanded((prev) => {
        const next = new Set(prev);
        if (next.has(folder.path)) {
          next.delete(folder.path);
          persistExpanded(folder.path, false);
        } else {
          next.add(folder.path);
          persistExpanded(folder.path, true);
        }
        return next;
      });
    },
    [forceExpand],
  );

  const handleSelectPlaybook = useCallback((pb: LibraryEntry) => {
    setSelection({ kind: 'playbook', playbook: pb });
  }, []);

  const selectedFile = selection?.kind === 'playbook' ? selection.playbook.file : null;
  const selectedFolderPath = selection?.kind === 'folder' ? selection.folder.path : null;

  const runImport = useCallback(
    async (playbook: LibraryEntry, remoteURL: string, runAfter: boolean): Promise<void> => {
      const result = await registerMutation.mutateAsync({
        url: remoteURL,
        name: playbook.name,
      });
      if (runAfter) {
        setRunTestId(result.test_id);
      }
    },
    [registerMutation],
  );

  const beginImport = useCallback(
    async (playbook: LibraryEntry, runAfter: boolean) => {
      if (!libraryQuery.data) return;
      setErrorMessage(null);
      setBusy(runAfter ? 'import-and-run' : 'import');
      try {
        const check = await checkMutation.mutateAsync(playbook.file);
        const remoteURL = libraryQuery.data.base_url + playbook.file;

        if (check.state === 'absent') {
          await runImport(playbook, remoteURL, runAfter);
        } else if (check.state === 'same') {
          // Already registered with identical YAML: skip re-register.
          if (runAfter && check.local_test_id) {
            setRunTestId(check.local_test_id);
          }
        } else {
          // Different — surface the overwrite warning before continuing.
          setOverwrite({ check, playbook, runAfter });
        }
      } catch (err) {
        setErrorMessage(err instanceof Error ? err.message : String(err));
      } finally {
        setBusy(null);
      }
    },
    [checkMutation, libraryQuery.data, runImport],
  );

  const confirmOverwrite = useCallback(async () => {
    if (!overwrite || !libraryQuery.data) return;
    const { playbook, runAfter } = overwrite;
    const remoteURL = libraryQuery.data.base_url + playbook.file;
    setOverwrite(null);
    setBusy(runAfter ? 'import-and-run' : 'import');
    try {
      await runImport(playbook, remoteURL, runAfter);
    } catch (err) {
      setErrorMessage(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(null);
    }
  }, [libraryQuery.data, overwrite, runImport]);

  if (libraryQuery.isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full size-8 border-b-2 border-primary-600" />
      </div>
    );
  }

  if (libraryQuery.error) {
    return (
      <div className="card p-6 text-center">
        <p className="text-error-600">
          Failed to load shared library: {libraryQuery.error.message}
        </p>
        <button onClick={() => libraryQuery.refetch()} className="btn btn-secondary btn-sm mt-4">
          Retry
        </button>
      </div>
    );
  }

  if (!libraryQuery.data || !filteredTree) {
    return null;
  }

  return (
    <div className="space-y-4">
      <div className="card p-3 flex flex-wrap items-center gap-3">
        <div className="flex items-center gap-2 flex-1 min-w-[16rem]">
          <input
            type="text"
            placeholder={
              scope === 'tags'
                ? 'Search by tag…'
                : scope === 'name'
                  ? 'Search by name or description…'
                  : 'Search by name, description, id, file or tag…'
            }
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="flex-1 px-3 py-1.5 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-xs text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          />
          <select
            value={scope}
            onChange={(e) => setScope(e.target.value as SearchScope)}
            className="px-2 py-1.5 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-xs text-xs"
            title="Search scope"
          >
            <option value="tags">tags</option>
            <option value="name">name + desc</option>
            <option value="all">all fields</option>
          </select>
        </div>

        <TagFilter all={allTags} selected={selectedTags} onChange={setSelectedTags} />

        <div className="text-xs text-[var(--color-text-tertiary)]">
          {libraryQuery.data.playbooks.length} playbooks
        </div>
      </div>

      {errorMessage && (
        <div className="card border border-error-300 dark:border-error-800 p-3 text-sm text-error-600 flex items-start justify-between gap-3">
          <span>{errorMessage}</span>
          <button onClick={() => setErrorMessage(null)} className="text-xs underline">
            dismiss
          </button>
        </div>
      )}

      <div className="card overflow-hidden">
        <SplitPane
          storageKey="library-split"
          defaultLeftWidth={35}
          minLeftWidth={20}
          maxLeftWidth={60}
          maxHeight="calc(100vh - 280px)"
          left={
            <LibraryTree
              root={filteredTree}
              selectedFile={selectedFile}
              selectedFolderPath={selectedFolderPath}
              registeredFiles={registeredFiles}
              expanded={expanded}
              forceExpand={forceExpand}
              onSelect={handleSelectPlaybook}
              onSelectFolder={handleSelectFolder}
            />
          }
          right={
            <LibraryDetails
              selection={selection}
              registered={
                selection?.kind === 'playbook'
                  ? registeredFiles.has(selection.playbook.file)
                  : false
              }
              canImport={isLoggedIn}
              baseURL={libraryQuery.data.base_url}
              busy={busy}
              onImport={(pb) => beginImport(pb, false)}
              onImportAndRun={(pb) => beginImport(pb, true)}
              onPlaybookSelect={handleSelectPlaybook}
            />
          }
        />
      </div>

      <StartTestModal
        isOpen={runTestId !== null}
        onClose={() => setRunTestId(null)}
        initialTestId={runTestId}
      />

      <OverwriteWarning
        open={overwrite !== null}
        check={overwrite?.check ?? null}
        onCancel={() => setOverwrite(null)}
        onConfirm={confirmOverwrite}
      />
    </div>
  );
}

// TagFilter is a small dropdown of chips. Clicking a tag toggles it;
// AND semantics are applied by the parent.
interface TagFilterProps {
  all: string[];
  selected: string[];
  onChange: (next: string[]) => void;
}

function TagFilter({ all, selected, onChange }: TagFilterProps) {
  const [open, setOpen] = useState(false);

  const toggle = (tag: string) => {
    if (selected.includes(tag)) {
      onChange(selected.filter((t) => t !== tag));
    } else {
      onChange([...selected, tag]);
    }
  };

  return (
    <div className="relative">
      <button
        onClick={() => setOpen((p) => !p)}
        className="btn btn-secondary btn-sm flex items-center gap-1.5"
      >
        Tags
        {selected.length > 0 && (
          <span className="px-1.5 py-0.5 bg-primary-600 text-white rounded-full text-xs">
            {selected.length}
          </span>
        )}
        <ChevronDownIcon className="size-3.5" />
      </button>
      {open && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setOpen(false)} />
          <div className="absolute right-0 mt-1 z-20 card p-2 max-h-72 overflow-auto min-w-[14rem]">
            {selected.length > 0 && (
              <div className="flex items-center justify-between px-2 py-1 mb-1 border-b border-[var(--color-border)]">
                <span className="text-xs text-[var(--color-text-secondary)]">
                  {selected.length} selected
                </span>
                <button
                  onClick={() => onChange([])}
                  className="text-xs text-primary-600 hover:underline"
                >
                  clear
                </button>
              </div>
            )}
            <div className="flex flex-wrap gap-1">
              {all.map((tag) => {
                const isSelected = selected.includes(tag);
                return (
                  <button
                    key={tag}
                    onClick={() => toggle(tag)}
                    className={`px-2 py-0.5 rounded-xs text-xs ${
                      isSelected
                        ? 'bg-primary-600 text-white'
                        : 'bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-secondary)]'
                    }`}
                  >
                    {tag}
                  </button>
                );
              })}
            </div>
          </div>
        </>
      )}
    </div>
  );
}

interface OverwriteWarningProps {
  open: boolean;
  check: LibraryCheckResponse | null;
  onCancel: () => void;
  onConfirm: () => void;
}

function OverwriteWarning({ open, check, onCancel, onConfirm }: OverwriteWarningProps) {
  return (
    <Modal isOpen={open} onClose={onCancel} title="Overwrite local copy?" size="lg">
      {check && (
        <div className="space-y-4">
          <p className="text-sm">
            A test with id <code className="font-mono">{check.remote_id}</code> is already
            registered locally with different YAML. Continuing will replace your local copy with
            the shared one.
          </p>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-3 text-xs">
            <div>
              <div className="font-medium mb-1 text-[var(--color-text-secondary)]">
                Local ({check.local_name})
              </div>
              <pre className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-xs p-2 overflow-auto max-h-64 whitespace-pre-wrap break-all">
                {check.local_yaml ?? ''}
              </pre>
            </div>
            <div>
              <div className="font-medium mb-1 text-[var(--color-text-secondary)]">
                Shared ({check.remote_name})
              </div>
              <pre className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-xs p-2 overflow-auto max-h-64 whitespace-pre-wrap break-all">
                {check.remote_yaml}
              </pre>
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <button onClick={onCancel} className="btn btn-secondary btn-sm">
              Cancel
            </button>
            <button onClick={onConfirm} className="btn btn-primary btn-sm">
              Overwrite & Continue
            </button>
          </div>
        </div>
      )}
    </Modal>
  );
}

function ChevronDownIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
    </svg>
  );
}

export default LibraryTab;
