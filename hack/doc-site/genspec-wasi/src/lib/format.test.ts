import { describe, expect, it } from 'vitest';
import { expectedBytes, humanBytes, humanDuration } from './format';

describe('humanDuration', () => {
  it('reads the way the terminal UI reads', () => {
    expect(humanDuration(0)).toBe('0ms');
    expect(humanDuration(947)).toBe('947ms');
    expect(humanDuration(3000)).toBe('3s');
    expect(humanDuration(63000)).toBe('1m 3s');
    expect(humanDuration(120000)).toBe('2m');
  });

  it('rounds rather than truncating, in both units', () => {
    expect(humanDuration(946.6)).toBe('947ms');
    expect(humanDuration(3400)).toBe('3s');
    expect(humanDuration(3600)).toBe('4s');
  });

  // The boundary is worth pinning: 999.6 ms must not render as "1000ms".
  it('crosses into seconds without a four-digit millisecond', () => {
    expect(humanDuration(999)).toBe('999ms');
    expect(humanDuration(999.6)).toBe('1s');
    expect(humanDuration(1000)).toBe('1s');
  });

  it('says nothing rather than something wrong', () => {
    expect(humanDuration(NaN)).toBe('—');
    expect(humanDuration(-1)).toBe('—');
  });
});

describe('humanBytes', () => {
  it('changes the unit rather than growing the number', () => {
    expect(humanBytes(512)).toBe('512 B');
    expect(humanBytes(2048)).toBe('2 KB');
    expect(humanBytes(5 * 1024 * 1024)).toBe('5.0 MB');
    expect(humanBytes(885 * 1024 * 1024)).toBe('885 MB');
  });

  // A tab lives or dies on the difference between 1.2 and 1.9 GB; nobody cares about 884 vs 885 MB.
  it('keeps two places in gigabytes and none in large megabytes', () => {
    expect(humanBytes(1024 * 1024 * 1024)).toBe('1.00 GB');
    expect(humanBytes(2.2 * 1024 * 1024 * 1024)).toBe('2.20 GB');
    expect(humanBytes(462 * 1024 * 1024)).toBe('462 MB');
  });

  it('says nothing rather than something wrong', () => {
    expect(humanBytes(NaN)).toBe('—');
    expect(humanBytes(-1)).toBe('—');
  });
});

describe('expectedBytes', () => {
  const headers = (h: Record<string, string>) => new Headers(h);

  it('trusts a plain length', () => {
    expect(expectedBytes(headers({ 'content-length': '20415084' }))).toBe(20415084);
  });

  // The bar would run past its end: Content-Length counts the wire, the stream delivers the decoded
  // bytes, and a bar that overshoots makes a reader distrust the rest of the page.
  it('refuses a length that describes a compressed body', () => {
    expect(expectedBytes(headers({ 'content-length': '3600000', 'content-encoding': 'gzip' })))
      .toBeUndefined();
    expect(expectedBytes(headers({ 'content-length': '3600000', 'content-encoding': 'br' })))
      .toBeUndefined();
  });

  it('has no answer when the response gives none', () => {
    expect(expectedBytes(headers({}))).toBeUndefined();
    expect(expectedBytes(headers({ 'content-length': '0' }))).toBeUndefined();
    expect(expectedBytes(headers({ 'content-length': 'lots' }))).toBeUndefined();
  });
});
