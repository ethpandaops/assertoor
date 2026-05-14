import type { LibraryFolder, LibraryEntry } from '../../types/api';

// TreeNode is a folder node in the library tree. Leaf playbooks live
// in `playbooks` on each node (they are NOT separate child nodes).
export interface TreeNode {
  path: string; // "" for the implicit root, otherwise "stable" / "pectra-dev/kurtosis" etc.
  name: string; // display name (from `_header.yaml` if available, else humanised)
  description?: string;
  playbooks: LibraryEntry[];
  children: TreeNode[];
}

export type SearchScope = 'tags' | 'name' | 'all';

export interface LibraryFilter {
  search: string;
  scope: SearchScope;
  tags: string[]; // AND semantics — playbook must have every selected tag
}

// buildTree groups the flat playbooks + folders lists from the index
// into a hierarchical tree mirroring the playbooks/ directory layout.
// Folder display names come from `folders` entries when present, else
// they fall back to a humanised version of the path segment.
export function buildTree(folders: LibraryFolder[], playbooks: LibraryEntry[]): TreeNode {
  const root: TreeNode = { path: '', name: '', playbooks: [], children: [] };
  const folderByPath = new Map<string, TreeNode>();
  folderByPath.set('', root);

  // Pre-create folder nodes from explicit `_header.yaml` entries so
  // their name / description are populated.
  for (const folder of folders) {
    ensureFolder(folderByPath, folder.path, folder.name, folder.description);
  }

  for (const pb of playbooks) {
    const parentPath = parentFolderOf(pb.file);
    const node = ensureFolder(folderByPath, parentPath);
    node.playbooks.push(pb);
  }

  // Stable order: folders alphabetical, playbooks alphabetical by name.
  sortTree(root);

  return root;
}

function ensureFolder(
  cache: Map<string, TreeNode>,
  path: string,
  name?: string,
  description?: string,
): TreeNode {
  const existing = cache.get(path);
  if (existing) {
    if (name && !existing.name) existing.name = name;
    if (description && !existing.description) existing.description = description;
    return existing;
  }

  const node: TreeNode = {
    path,
    name: name ?? humaniseSegment(lastSegment(path)),
    description,
    playbooks: [],
    children: [],
  };
  cache.set(path, node);

  if (path !== '') {
    const parent = ensureFolder(cache, parentPathOf(path));
    parent.children.push(node);
  }

  return node;
}

function parentFolderOf(file: string): string {
  const idx = file.lastIndexOf('/');
  return idx < 0 ? '' : file.slice(0, idx);
}

function parentPathOf(path: string): string {
  const idx = path.lastIndexOf('/');
  return idx < 0 ? '' : path.slice(0, idx);
}

function lastSegment(path: string): string {
  const idx = path.lastIndexOf('/');
  return idx < 0 ? path : path.slice(idx + 1);
}

function humaniseSegment(segment: string): string {
  if (!segment) return '';
  return segment
    .split(/[-_]/)
    .map((part) => (part ? part[0].toUpperCase() + part.slice(1) : ''))
    .join(' ');
}

function sortTree(node: TreeNode): void {
  node.children.sort((a, b) => a.path.localeCompare(b.path));
  node.playbooks.sort((a, b) => a.name.localeCompare(b.name));
  for (const child of node.children) {
    sortTree(child);
  }
}

// matchesFilter returns true when a playbook satisfies the active
// filter. Tags use AND semantics. The text search is applied against
// the configured scope.
export function matchesFilter(pb: LibraryEntry, filter: LibraryFilter): boolean {
  if (filter.tags.length > 0) {
    const pbTags = pb.tags ?? [];
    for (const required of filter.tags) {
      if (!pbTags.includes(required)) return false;
    }
  }

  const query = filter.search.trim().toLowerCase();
  if (query === '') return true;

  const haystack = buildHaystack(pb, filter.scope);
  return haystack.includes(query);
}

function buildHaystack(pb: LibraryEntry, scope: SearchScope): string {
  const tags = (pb.tags ?? []).join(' ');
  switch (scope) {
    case 'tags':
      return tags.toLowerCase();
    case 'name':
      return [pb.name, pb.description ?? ''].join(' ').toLowerCase();
    case 'all':
    default:
      return [pb.id, pb.file, pb.name, pb.description ?? '', tags].join(' ').toLowerCase();
  }
}

// filterTree returns a new tree containing only the playbooks that
// match the filter. Folders with no matching playbooks (directly or in
// descendants) are pruned. Each node's `playbooks` is filtered in-place
// to the matching subset.
export function filterTree(node: TreeNode, filter: LibraryFilter): TreeNode | null {
  const filteredChildren: TreeNode[] = [];
  for (const child of node.children) {
    const f = filterTree(child, filter);
    if (f) filteredChildren.push(f);
  }

  const filteredPlaybooks = node.playbooks.filter((pb) => matchesFilter(pb, filter));

  if (filteredPlaybooks.length === 0 && filteredChildren.length === 0 && node.path !== '') {
    return null;
  }

  return {
    ...node,
    playbooks: filteredPlaybooks,
    children: filteredChildren,
  };
}

// collectTags returns the unique tag set used across all playbooks,
// sorted alphabetically. Useful for populating the tag filter UI.
export function collectTags(playbooks: LibraryEntry[]): string[] {
  const set = new Set<string>();
  for (const pb of playbooks) {
    for (const tag of pb.tags ?? []) {
      set.add(tag);
    }
  }
  return Array.from(set).sort((a, b) => a.localeCompare(b));
}

// countPlaybooks reports how many playbooks (direct + descendant) are
// present in a node. Used for the count badge in the tree.
export function countPlaybooks(node: TreeNode): number {
  let total = node.playbooks.length;
  for (const child of node.children) {
    total += countPlaybooks(child);
  }
  return total;
}
