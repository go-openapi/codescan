import { describe, expect, it } from 'vitest';
import { argvFor } from './flags';
import { defaultOptions } from './types';

describe('argvFor', () => {
  it('asks for the machine-readable result', () => {
    // The bare document carries no positions, so an editor has nothing to mark and the two panes
    // have nothing to join on.
    expect(argvFor(defaultOptions, 'embedded')).toContain('-format=json');
  });

  it('always spells out the loader and the build target', () => {
    // A guest has no toolchain, and left alone would describe the platform it is running on rather
    // than the code being scanned.
    const argv = argvFor(defaultOptions, 'embedded');

    expect(argv).toContain('-loader=own');
    expect(argv.join(' ')).toContain('-goos linux');
    expect(argv.join(' ')).toContain('-goarch amd64');
  });

  it('renders booleans as -name=value so a true-by-default flag can be turned off', () => {
    const argv = argvFor({ ...defaultOptions, scanModels: false }, 'embedded');

    expect(argv).toContain('-scan-models=false');
    expect(argv).not.toContain('-scan-models');
  });

  it('passes each option through under its real flag name', () => {
    const argv = argvFor({ ...defaultOptions, pruneUnusedModels: true, refAliases: true }, 'embedded');

    expect(argv).toContain('-prune-unused-models=true');
    expect(argv).toContain('-ref-aliases=true');
  });

  it('says nothing about the standard library when the artifact carries its own copy', () => {
    // The embedded archive is picked up by the command itself. Naming a flag here would either
    // point at a path that is not mounted, or turn on the degraded mode we are trying to leave.
    const argv = argvFor(defaultOptions, 'embedded').join(' ');

    expect(argv).not.toContain('-stub-stdlib');
    expect(argv).not.toContain('-export-data');
  });

  it('names the archive when the host mounted one, and synthesizes only when asked', () => {
    expect(argvFor(defaultOptions, 'mounted').join(' ')).toContain('-export-data=');
    expect(argvFor(defaultOptions, 'stub')).toContain('-stub-stdlib=true');
  });

  it('omits build tags when there are none, rather than passing an empty one', () => {
    expect(argvFor(defaultOptions, 'embedded')).not.toContain('-build-tags');
    expect(argvFor({ ...defaultOptions, buildTags: ' integration ' }, 'embedded')).toContain('integration');
  });

  it('ends with the working directory and the pattern', () => {
    expect(argvFor(defaultOptions, 'embedded').slice(-3)).toEqual(['-workdir', '/src', './...']);
  });
});
