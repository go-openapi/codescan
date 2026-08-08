import type { ScanOptions } from './types';

// The flag each option maps to. These are genspec-wasi's real flag names - check `genspec-wasi -h`, or
// cmd/genspec-wasi/options.go, before adding one: a name that does not exist fails the whole scan with
// "flag provided but not defined".
const booleans: Array<[keyof ScanOptions, string]> = [
  ['scanModels', 'scan-models'],
  ['pruneUnusedModels', 'prune-unused-models'],
  ['refAliases', 'ref-aliases'],
  ['transparentAliases', 'transparent-aliases'],
  ['setXNullableForPointers', 'set-x-nullable-for-pointers'],
  ['skipExtensions', 'skip-extensions'],
];

// argvFor renders the command line the guest sees. The guest has no toolchain and nothing mounted
// beyond the module under scan, so the loader and the build target are always spelled out.
//
// Booleans go as -name=value rather than bare -name: scan-models defaults to true, and Go's flag
// package needs the explicit form to turn one off.
// stdlib says where the standard library's types come from. 'embedded' is the artifact's own copy,
// carried by the exportdata build tag and picked up with no flag at all; 'mounted' reads an archive
// the host put in the guest filesystem. 'stub' synthesizes from usage - it needs no asset, and it
// loses method sets, so it is the degraded mode rather than a choice.
export function argvFor(options: ScanOptions, stdlib: 'embedded' | 'mounted' | 'stub'): string[] {
  // -format=json is not optional here: positions are what the editor draws marks from and what
  // cross-references join on, and the bare document carries neither.
  const argv = ['genspec-wasi.wasm', '-format=json', '-loader=own', '-goos', 'linux', '-goarch', 'amd64'];

  if (stdlib === 'mounted') {
    argv.push('-export-data=/exportdata.zip');
  } else if (stdlib === 'stub') {
    argv.push('-stub-stdlib=true');
  }

  for (const [key, flag] of booleans) {
    argv.push(`-${flag}=${options[key] ? 'true' : 'false'}`);
  }
  if (options.buildTags.trim()) {
    argv.push('-build-tags', options.buildTags.trim());
  }
  argv.push('-workdir', '/src', './...');

  return argv;
}
