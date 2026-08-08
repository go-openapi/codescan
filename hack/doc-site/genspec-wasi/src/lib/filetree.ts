// Turning a flat list of paths into something navigable.
//
// A dropdown is fine for the three-file sample and untenable for a vendored module: dockerctl is
// 1,490 files, and 502 of them are its own. What makes a tree readable at that size is not the tree
// but the collapsing - a run of directories with one child each is one step in the mind and should
// be one row on screen.

export type TreeNode = {
  /** What to show. A collapsed chain shows as "github.com/go-openapi/swag". */
  name: string;
  /** Full path: the file's own, or the directory prefix ending in "/". */
  path: string;
  dir: boolean;
  children: TreeNode[];
};

type Building = {
  name: string;
  dirs: Map<string, Building>;
  files: string[];
};

function emptyDir(name: string): Building {
  return { name, dirs: new Map(), files: [] };
}

// build assembles the tree, directories before files and each sorted by name - the order a file
// manager uses, and the one a reader scanning for a package expects.
export function build(paths: string[]): TreeNode[] {
  const root = emptyDir('');

  for (const path of paths) {
    const parts = path.split('/').filter(Boolean);
    const file = parts.pop();
    if (!file) {
      continue;
    }

    let at = root;
    for (const part of parts) {
      let next = at.dirs.get(part);
      if (!next) {
        next = emptyDir(part);
        at.dirs.set(part, next);
      }
      at = next;
    }
    at.files.push(file);
  }

  return childrenOf(root, '');
}

function childrenOf(node: Building, prefix: string): TreeNode[] {
  const dirs = [...node.dirs.values()]
    .sort((a, b) => a.name.localeCompare(b.name))
    .map((child) => collapse(child, prefix));

  const files = [...node.files]
    .sort((a, b) => a.localeCompare(b))
    .map((name) => ({ name, path: prefix + name, dir: false, children: [] }));

  return [...dirs, ...files];
}

// collapse folds a chain of single-child directories into one row.
//
// vendor/github.com/go-openapi/swag/mangling is five levels holding nothing but each other. Shown
// one per row it is five rows of scrolling to reach a package name; shown as one it is a path, which
// is how it is spoken about anyway.
function collapse(node: Building, prefix: string): TreeNode {
  let name = node.name;
  let at = node;
  while (at.files.length === 0 && at.dirs.size === 1) {
    const only = [...at.dirs.values()][0];
    name += '/' + only.name;
    at = only;
  }

  const path = prefix + name + '/';

  return { name, path, dir: true, children: childrenOf(at, path) };
}

// flatten walks the tree in display order, keeping only what is open.
//
// The visible rows are what the arrow keys move through and what gets rendered, so both work from
// one list rather than each re-deriving what is on screen.
export function flatten(
  nodes: TreeNode[],
  open: (path: string) => boolean,
  depth = 0,
): Array<{ node: TreeNode; depth: number }> {
  const rows: Array<{ node: TreeNode; depth: number }> = [];
  for (const node of nodes) {
    rows.push({ node, depth });
    if (node.dir && open(node.path)) {
      rows.push(...flatten(node.children, open, depth + 1));
    }
  }

  return rows;
}

// pathsToOpen returns the directories that must be open for a file to be visible.
export function pathsToOpen(nodes: TreeNode[], file: string): string[] {
  const opened: string[] = [];

  const walk = (list: TreeNode[]): boolean => {
    for (const node of list) {
      if (!node.dir) {
        if (node.path === file) {
          return true;
        }
        continue;
      }
      if (file.startsWith(node.path) && walk(node.children)) {
        opened.push(node.path);

        return true;
      }
    }

    return false;
  };

  walk(nodes);

  // Root first. The caller only unions them into a set, but a list that reads outside-in is the one
  // you can check by eye against the tree.
  return opened.reverse();
}
