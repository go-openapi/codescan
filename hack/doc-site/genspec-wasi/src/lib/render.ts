// Rendering the document so that every node can be found again.
//
// JSON.stringify would produce the same text, and nothing else. What the panes need is the inverse
// map - given an RFC 6901 pointer, which lines is that node written on - because that is what the
// scanner's provenance is keyed by, and therefore the only way a spec node can be tied back to the
// Go source that produced it.
//
// So the document is walked once, emitting lines and recording a span as each node closes. The same
// walk renders YAML, which is what keeps the format toggle from quietly dropping cross-references.

export type Span = {
  /** First line the node occupies, 1-based. */
  from: number;
  /** Last line, inclusive. A scalar starts and ends on the same one. */
  to: number;
};

export type Rendered = {
  text: string;
  /** RFC 6901 pointer → the lines it occupies. The root is "". */
  spans: Map<string, Span>;
};

export type Format = 'json' | 'yaml';

export function render(value: unknown, format: Format): Rendered {
  return format === 'yaml' ? renderYAML(value) : renderJSON(value);
}

// escapeToken escapes one path segment per RFC 6901: ~ becomes ~0, / becomes ~1.
//
// The order matters and is not symmetric with unescaping - do / first and a segment containing ~1
// would round-trip wrong.
export function escapeToken(segment: string): string {
  return segment.replace(/~/g, '~0').replace(/\//g, '~1');
}

export function unescapeToken(segment: string): string {
  return segment.replace(/~1/g, '/').replace(/~0/g, '~');
}

export function pointerTo(parent: string, segment: string | number): string {
  return `${parent}/${typeof segment === 'number' ? segment : escapeToken(segment)}`;
}

// parentOf returns the pointer one level up, or null at the root.
export function parentOf(pointer: string): string | null {
  if (!pointer) {
    return null;
  }
  const cut = pointer.lastIndexOf('/');

  return cut <= 0 ? '' : pointer.slice(0, cut);
}

class Writer {
  lines: string[] = [];
  spans = new Map<string, Span>();

  get line(): number {
    return this.lines.length;
  }

  push(text: string) {
    this.lines.push(text);
  }

  // append adds to the line already open, for a scalar written after its key.
  append(text: string) {
    this.lines[this.lines.length - 1] += text;
  }

  record(pointer: string, from: number, to: number) {
    this.spans.set(pointer, { from, to });
  }

  done(): Rendered {
    return { text: this.lines.join('\n'), spans: this.spans };
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

// ---- JSON ----------------------------------------------------------------

function renderJSON(value: unknown): Rendered {
  const w = new Writer();
  w.push('');
  writeJSON(w, value, '', '', '');
  w.record('', 1, w.line);

  return w.done();
}

// writeJSON writes value onto the currently open line, opening more as needed, and records a span
// for every node beneath it.
//
// prefix is what has already been written on the open line (a key, or an indent); trailer is what
// follows the value (a comma). Both are the caller's business because only the caller knows whether
// this is the last member.
function writeJSON(w: Writer, value: unknown, pointer: string, indent: string, trailer: string) {
  const start = w.line;

  if (Array.isArray(value)) {
    if (value.length === 0) {
      w.append(`[]${trailer}`);
    } else {
      w.append('[');
      const inner = indent + '  ';
      value.forEach((item, i) => {
        w.push(inner);
        writeJSON(w, item, pointerTo(pointer, i), inner, i === value.length - 1 ? '' : ',');
      });
      w.push(`${indent}]${trailer}`);
    }
  } else if (isRecord(value)) {
    const keys = Object.keys(value);
    if (keys.length === 0) {
      w.append(`{}${trailer}`);
    } else {
      w.append('{');
      const inner = indent + '  ';
      keys.forEach((key, i) => {
        w.push(`${inner}${JSON.stringify(key)}: `);
        writeJSON(w, value[key], pointerTo(pointer, key), inner, i === keys.length - 1 ? '' : ',');
      });
      w.push(`${indent}}${trailer}`);
    }
  } else {
    w.append(`${JSON.stringify(value)}${trailer}`);
  }

  if (pointer) {
    w.record(pointer, start, w.line);
  }
}

// ---- YAML ----------------------------------------------------------------

// plainSafe reports whether a string can be written bare.
//
// Deliberately strict. Anything doubtful is double-quoted, which is always valid - YAML's
// double-quoted scalar takes JSON's escapes - so being wrong here costs quote marks rather than a
// document that parses back as something else. The cases that matter: a string that looks like a
// number, a boolean or null; one that starts with an indicator character; one carrying a colon or a
// hash, which would start a mapping or a comment; and anything with a newline.
const yamlIndicators = /^[-?:,[\]{}#&*!|>'"%@`]/;
const yamlLooksScalar = /^(?:true|false|null|~|[-+]?\d+(?:\.\d+)?(?:[eE][-+]?\d+)?)$/i;

function plainSafe(s: string): boolean {
  if (s === '' || s !== s.trim() || yamlIndicators.test(s) || yamlLooksScalar.test(s)) {
    return false;
  }

  return !/[:#\n\r\t]/.test(s) && !s.includes(': ') && !s.includes(' #');
}

function yamlScalar(value: unknown): string {
  if (value === null || value === undefined) {
    return 'null';
  }
  if (typeof value === 'string') {
    return plainSafe(value) ? value : JSON.stringify(value);
  }

  return String(value);
}

function renderYAML(value: unknown): Rendered {
  const w = new Writer();

  // A document that is not a mapping has nothing to hang keys off, so it is written as one scalar.
  if (!isRecord(value) && !Array.isArray(value)) {
    w.push(yamlScalar(value));
    w.record('', 1, 1);

    return w.done();
  }

  writeYAMLBody(w, value, '', '');
  w.record('', 1, Math.max(1, w.line));

  return w.done();
}

// writeYAMLBody writes a container's members, each on its own line, at the given indent.
function writeYAMLBody(w: Writer, value: unknown, pointer: string, indent: string) {
  if (Array.isArray(value)) {
    for (const [i, item] of value.entries()) {
      writeYAMLMember(w, item, pointerTo(pointer, i), indent, '- ');
    }

    return;
  }

  for (const [key, item] of Object.entries(value as Record<string, unknown>)) {
    writeYAMLMember(w, item, pointerTo(pointer, key), indent, `${yamlKey(key)}: `);
  }
}

function yamlKey(key: string): string {
  return plainSafe(key) ? key : JSON.stringify(key);
}

// writeYAMLMember writes one member: its lead ("- " or "key: ") and then its value, which either
// fits on that line or opens a block beneath it.
function writeYAMLMember(w: Writer, value: unknown, pointer: string, indent: string, lead: string) {
  const start = w.line + 1;
  const empty = (Array.isArray(value) && value.length === 0)
    || (isRecord(value) && Object.keys(value).length === 0);

  if (empty) {
    w.push(`${indent}${lead}${Array.isArray(value) ? '[]' : '{}'}`);
  } else if (Array.isArray(value) || isRecord(value)) {
    w.push(`${indent}${lead.trimEnd()}`);
    // A sequence sits at its parent's indent - its "- " is the indent - while a mapping steps in.
    writeYAMLBody(w, value, pointer, Array.isArray(value) ? indent : indent + '  ');
  } else {
    w.push(`${indent}${lead}${yamlScalar(value)}`);
  }

  w.record(pointer, start, w.line);
}
