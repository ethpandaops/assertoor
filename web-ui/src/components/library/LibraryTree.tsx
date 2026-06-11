import type { LibraryEntry } from '../../types/api';
import { countPlaybooks, type TreeNode } from './libraryTreeUtils';

interface LibraryTreeProps {
  // The (already filtered) root node. Direct playbooks at root are
  // rendered above the folder list.
  root: TreeNode;
  selectedFile: string | null;
  selectedFolderPath: string | null;
  // Set of file paths already registered locally — used to show a
  // small badge next to those playbook rows.
  registeredFiles: Set<string>;
  // Set of folder paths currently expanded. Owned by the parent so
  // the state survives filter / re-render cycles and so the parent
  // can force-open folders that have matches.
  expanded: Set<string>;
  // Set of folder paths that should be forced-open right now (used by
  // the search UX to auto-expand folders containing matches). Forced
  // folders cannot be collapsed by the user while the force is active.
  forceExpand: Set<string>;
  onSelect: (pb: LibraryEntry) => void;
  // Selecting a folder shows its details in the right pane AND toggles
  // its expand state in one click.
  onSelectFolder: (folder: TreeNode) => void;
}

function LibraryTree({
  root,
  selectedFile,
  selectedFolderPath,
  registeredFiles,
  expanded,
  forceExpand,
  onSelect,
  onSelectFolder,
}: LibraryTreeProps) {
  const hasContent = root.children.length > 0 || root.playbooks.length > 0;
  if (!hasContent) {
    return (
      <div className="p-6 text-sm text-[var(--color-text-secondary)]">
        No playbooks match your filters.
      </div>
    );
  }

  return (
    <ul className="text-sm py-1">
      {root.playbooks.map((pb) => (
        <PlaybookRow
          key={pb.file}
          playbook={pb}
          depth={0}
          selected={pb.file === selectedFile}
          registered={registeredFiles.has(pb.file)}
          onSelect={onSelect}
        />
      ))}
      {root.children.map((node) => (
        <FolderRow
          key={node.path}
          node={node}
          depth={0}
          selectedFile={selectedFile}
          selectedFolderPath={selectedFolderPath}
          registeredFiles={registeredFiles}
          expanded={expanded}
          forceExpand={forceExpand}
          onSelect={onSelect}
          onSelectFolder={onSelectFolder}
        />
      ))}
    </ul>
  );
}

interface FolderRowProps {
  node: TreeNode;
  depth: number;
  selectedFile: string | null;
  selectedFolderPath: string | null;
  registeredFiles: Set<string>;
  expanded: Set<string>;
  forceExpand: Set<string>;
  onSelect: (pb: LibraryEntry) => void;
  onSelectFolder: (folder: TreeNode) => void;
}

function FolderRow({
  node,
  depth,
  selectedFile,
  selectedFolderPath,
  registeredFiles,
  expanded,
  forceExpand,
  onSelect,
  onSelectFolder,
}: FolderRowProps) {
  const forced = forceExpand.has(node.path);
  const isOpen = forced || expanded.has(node.path);
  const isSelected = selectedFolderPath === node.path;
  const total = countPlaybooks(node);

  return (
    <li>
      <button
        type="button"
        onClick={() => onSelectFolder(node)}
        className={`w-full flex items-center justify-between gap-2 px-2 py-1.5 text-left rounded-xs ${
          isSelected
            ? 'bg-primary-50 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300'
            : 'hover:bg-[var(--color-bg-tertiary)]'
        }`}
        style={{ paddingLeft: `${depth * 16 + 8}px` }}
        aria-expanded={isOpen}
      >
        <span className="flex items-center gap-1.5 min-w-0">
          <ChevronIcon
            className={`size-3.5 flex-shrink-0 transition-transform ${isOpen ? 'rotate-90' : ''}`}
          />
          <span className="font-medium truncate">{node.name || node.path}</span>
        </span>
        <span className="text-xs text-[var(--color-text-tertiary)] flex-shrink-0">{total}</span>
      </button>

      {isOpen && (
        <ul>
          {node.playbooks.map((pb) => (
            <PlaybookRow
              key={pb.file}
              playbook={pb}
              depth={depth + 1}
              selected={pb.file === selectedFile}
              registered={registeredFiles.has(pb.file)}
              onSelect={onSelect}
            />
          ))}
          {node.children.map((child) => (
            <FolderRow
              key={child.path}
              node={child}
              depth={depth + 1}
              selectedFile={selectedFile}
              selectedFolderPath={selectedFolderPath}
              registeredFiles={registeredFiles}
              expanded={expanded}
              forceExpand={forceExpand}
              onSelect={onSelect}
              onSelectFolder={onSelectFolder}
            />
          ))}
        </ul>
      )}
    </li>
  );
}

interface PlaybookRowProps {
  playbook: LibraryEntry;
  depth: number;
  selected: boolean;
  registered: boolean;
  onSelect: (pb: LibraryEntry) => void;
}

function PlaybookRow({ playbook, depth, selected, registered, onSelect }: PlaybookRowProps) {
  const isDev = (playbook.version ?? '').startsWith('0.');

  return (
    <li>
      <button
        type="button"
        onClick={() => onSelect(playbook)}
        className={`w-full flex items-center gap-2 px-2 py-1.5 text-left rounded-xs ${
          selected
            ? 'bg-primary-50 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300'
            : 'hover:bg-[var(--color-bg-tertiary)]'
        }`}
        style={{ paddingLeft: `${depth * 16 + 24}px` }}
      >
        <span
          className={`size-2 rounded-full flex-shrink-0 ${
            isDev ? 'bg-yellow-500' : 'bg-emerald-500'
          }`}
          title={isDev ? 'In development (0.x)' : 'Production-ready (1.x+)'}
        />
        <span className="truncate flex-1">{playbook.name}</span>
        {registered && (
          <span
            className="text-xs text-primary-600 flex-shrink-0"
            title="Already registered locally"
          >
            ●
          </span>
        )}
      </button>
    </li>
  );
}

function ChevronIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
    </svg>
  );
}

export default LibraryTree;
