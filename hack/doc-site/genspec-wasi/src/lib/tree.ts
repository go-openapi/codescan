import type { SourceFile } from './types';

// wanted decides whether a picked file is worth carrying into the guest.
//
// Tests are never scanned. vendor/ is kept: with no module cache to read, a vendored tree is how a
// third-party import resolves - including the ones whose meaning lives in comments, which export data
// can never carry.
//
// vendor/modules.txt is kept for that same reason and is the easiest thing here to lose, being the one
// file that is not Go. The loader treats a vendor directory as authoritative only when it can read
// that file (internal/packages/list/resolve.go, vendorMode). Dropping it does not fail the scan: the
// whole vendored tree is ignored and every dependency is synthesized instead, which surfaces as a wall
// of scan.synthesized-import warnings that say nothing about the actual mistake.
export function wanted(path: string): boolean {
  if (path === 'vendor/modules.txt' || path.endsWith('/vendor/modules.txt')) {
    return true;
  }

  const name = path.slice(path.lastIndexOf('/') + 1);
  if (name === 'go.mod') {
    return true;
  }

  return name.endsWith('.go') && !name.endsWith('_test.go');
}

// reroot makes paths relative to the module root, and drops anything outside it.
//
// A directory pick reports paths relative to the chosen folder, which may sit above the module - or
// below it, if someone picks a subdirectory. The scan needs go.mod at the top, so find it and move
// everything to match. With no go.mod at all the paths are left alone and the scan will say so.
export function reroot(files: SourceFile[]): SourceFile[] {
  const mods = files.filter((f) => f.path === 'go.mod' || f.path.endsWith('/go.mod'));
  if (!mods.length) {
    return files;
  }

  // The outermost go.mod wins: a nested one belongs to a submodule the scan should not be rooted at.
  const root = mods
    .map((f) => f.path.slice(0, Math.max(0, f.path.length - 'go.mod'.length)))
    .reduce((a, b) => (a.length <= b.length ? a : b));

  return files
    .filter((f) => f.path.startsWith(root))
    .map((f) => ({ ...f, path: f.path.slice(root.length) }));
}

// What a selection may weigh.
//
// Everything is held in memory and handed to the guest on every scan, so this is a real ceiling and
// not a formality. It is set from measurement rather than taste: codescan itself vendors to 3.5 MB
// across 272 Go files, and go-swagger's dockerctl - a generated API client, which is the shape a
// visitor is most likely to arrive with - to 25 MB across some 1,490. A limit under that rejects the
// exact case the playground exists to serve.
//
// The warning band is where a scan stops being instant rather than where it stops working.
export const limits = {
  warnBytes: 16 * 1024 * 1024,
  maxBytes: 64 * 1024 * 1024,
};

export type Picked = { path: string; size: number };

export type Budget = {
  files: number;
  bytes: number;
  /** Over the hard ceiling: the selection is refused. */
  refused: boolean;
  /** Over the warning band: it will work, and it will not be quick. */
  heavy: boolean;
  /** The directory holding most of the weight, named so a refusal can point at something. */
  heaviest: { path: string; bytes: number } | null;
};

// weigh tallies a selection and names where its weight is.
//
// "Too big, pick something smaller" is not advice when the thing you picked is a module: the answer
// is nearly always a directory inside it, and the user cannot see which without being told.
export function weigh(picked: Picked[]): Budget {
  let bytes = 0;
  const byDir = new Map<string, number>();

  for (const entry of picked) {
    bytes += entry.size;

    // The first segment is the directory that was picked, so it names everything and distinguishes
    // nothing. Group one level below it, which is where vendor/ shows up.
    const parts = entry.path.split('/');
    const dir = parts.length > 2 ? parts.slice(0, 2).join('/') : parts[0];
    byDir.set(dir, (byDir.get(dir) ?? 0) + entry.size);
  }

  let heaviest: Budget['heaviest'] = null;
  for (const [path, size] of byDir) {
    if (!heaviest || size > heaviest.bytes) {
      heaviest = { path, bytes: size };
    }
  }

  return {
    files: picked.length,
    bytes,
    refused: bytes > limits.maxBytes,
    heavy: bytes > limits.warnBytes,
    heaviest,
  };
}


// needsVendoring reports a module that names dependencies and brought none of their source.
//
// Answerable from go.mod alone, before a scan runs, and worth answering there: the alternative is a
// wall of synthesized-import warnings that describe the consequence rather than the cause. There is
// no module cache in the guest, so a dependency's types - and any annotation in its comments -
// can only arrive as source.
//
// Deliberately a shallow read. Parsing go.mod properly would answer the same question no better: a
// require directive at the start of a line is what the go command writes, and anything cleverer
// would only differ on files the go command itself would reject.
export function needsVendoring(files: SourceFile[]): boolean {
  const mod = files.find((f) => f.path === 'go.mod');
  if (!mod || files.some((f) => f.path === 'vendor/modules.txt')) {
    return false;
  }

  return /^\s*require\s/m.test(mod.text);
}
