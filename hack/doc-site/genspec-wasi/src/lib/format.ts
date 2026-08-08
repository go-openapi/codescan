// Rendering numbers for the status line: how long, and how much.
//
// One home for both, and one humanBytes - there were briefly two, differing only in their
// rounding, which is exactly how a status line ends up disagreeing with itself.

// humanDuration renders a duration the way genspec-tui's status line does: "947ms", "3s", "1m 3s",
// and "2m" when the seconds are zero.
//
// Ported rather than reinvented so the two front-ends read the same. Milliseconds below a second,
// then whole seconds - a scan is never interesting to a tenth of a second, and a jittering last
// digit reads as instability.
export function humanDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) {
    return '—';
  }
  // Rounded before the comparison, not after: 999.6 ms is under a second and rounds to 1000, which
  // would print as a four-digit millisecond rather than as "1s".
  const rounded = Math.round(ms);
  if (rounded < 1000) {
    return `${rounded}ms`;
  }

  const seconds = Math.round(ms / 1000);
  if (seconds < 60) {
    return `${seconds}s`;
  }

  const minutes = Math.floor(seconds / 60);
  const rest = seconds % 60;

  return rest === 0 ? `${minutes}m` : `${minutes}m ${rest}s`;
}

// humanBytes renders a size the way the status line wants it: a unit that changes rather than a
// number that grows, and more precision the larger it gets - a browser tab lives or dies on the
// difference between 1.2 and 1.9 GB, and nobody cares whether a heap is 884 or 885 MB.
export function humanBytes(n: number): string {
  if (!Number.isFinite(n) || n < 0) {
    return '—';
  }
  if (n < 1024) {
    return `${Math.round(n)} B`;
  }
  if (n < 1024 * 1024) {
    return `${Math.round(n / 1024)} KB`;
  }
  if (n < 1024 * 1024 * 1024) {
    const mb = n / 1024 / 1024;

    return `${mb < 10 ? mb.toFixed(1) : Math.round(mb)} MB`;
  }

  return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

// expectedBytes reports how many bytes a response body will deliver, when that can be trusted.
//
// Content-Length counts the bytes on the wire. A compressed body delivers more than that once
// decoded, so a progress bar built on it would run past its end - and a bar that overshoots is worse
// than one that never claimed to know, because it makes the reader distrust the rest of the page.
export function expectedBytes(headers: Headers): number | undefined {
  if (headers.get('content-encoding')) {
    return undefined;
  }

  const declared = Number(headers.get('content-length') ?? 0);

  return Number.isFinite(declared) && declared > 0 ? declared : undefined;
}
