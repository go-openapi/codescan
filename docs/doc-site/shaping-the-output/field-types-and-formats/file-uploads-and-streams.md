---
title: File uploads and byte streams
weight: 15
description: |
  How io.Reader, multipart.File and the other stream types render — type: file
  on a formData parameter, base64 bytes everywhere else — and how to say what
  the bytes actually are.
---

A Go type like `io.Reader` says that bytes will flow. It says nothing about
*what* they are, how they are framed, or how long they run. codescan recognizes
these types and answers with the only two things Swagger 2.0 lets it say about
opaque bytes — picked by **where the field sits**, not by anything in the
declaration.

## The two answers

| Position | Rendering |
|---|---|
| `in: formData` parameter | `type: file` |
| model field, body, response body, header, other parameters | `{type: string, format: byte}` |

`type: file` is the canonical upload shape, and formData is the only location
Swagger 2.0 permits it in. Everywhere else the bytes travel inside a JSON
document, which cannot carry raw octets — so they render as `format: byte`, the
base64-encoded string the specification defines for exactly this.

## Uploading a file

Put the stream in a `formData` parameter and consume `multipart/form-data`:

{{< example go="shaping/streams/streams.go" goregion="params" golabel="parameters"
            json="shaping/streams/testdata/upload_params.json" jsonlabel="parameters" >}}

`upload` becomes `type: file`; the sibling `caption` is an ordinary form field.
`multipart.File` and `io.Reader` are interchangeable here — both are recognized.

## Streams in a model or a body

Anywhere that is not a formData parameter, the same types render as base64
bytes:

{{< example go="shaping/streams/streams.go" goregion="model" golabel="model"
            json="shaping/streams/testdata/attachment.json" jsonlabel="#/definitions/Attachment" >}}

`content` and `thumbnail` carry `{string, byte}`. `checksum` carries
`swagger:strfmt base64`, and the annotation wins — which is the point of the
next section.

## Say what the bytes are

The default is deliberately uninformative, because a stream *is*
uninformative. When you know more, say so and codescan will step aside:

- [`swagger:strfmt`]({{% relref "forcing-a-format" %}}) — name the format
  (`base64`, `binary`, a custom one);
- [`swagger:type`]({{% relref "/maintainers/annotations/swagger-type" %}}) —
  override the type outright;
- [`swagger:file`]({{% relref "/maintainers/annotations/swagger-file" %}}) —
  force the file shape where you want it and the position allows it.

## What is recognized

| Package | Types |
|---|---|
| `io` | `Reader`, `ReadCloser`, `ReadSeeker`, `ReadSeekCloser`, `ReadWriter`, `ReaderAt`, `ReaderFrom`, `LimitedReader`, `ByteReader`, `ByteScanner` |
| `mime/multipart` | `File` |
| `github.com/go-openapi/runtime` | `NamedReadCloser` |

Recognition is by **identity** — the exact named type — never by shape. An
interface of your own that happens to have a `Read` method is *your* type and is
documented as you declared it.

Because both renderings erase *which* stream it was — every type in the table
produces the same schema — each one also carries an
[`x-go-type`]({{% relref "vendor-extensions" %}}) extension naming the Go type it
came from, so a consumer can tell an `io.Reader` field from a `multipart.File`
one. `SkipExtensions` suppresses it along with the rest of the `x-go-*` family.

{{% notice style="note" %}}
**`io.Writer` is not recognized**, nor are the write-only closers. A sink the
caller writes into is not something that travels on the wire, so codescan does
not assume what you meant by putting one in an API type — it documents the type
structurally, and you override it if you had something in mind.
{{% /notice %}}

## What's next

- [Forcing a conformant format]({{% relref "forcing-a-format" %}}) — the
  field-level `swagger:strfmt` used above.
- [Type discovery]({{% relref "/shaping-the-output/scope-and-discovery/type-discovery" %}}) —
  how codescan decides what a Go type becomes when no recognizer applies.
- [`swagger:file` reference]({{% relref "/maintainers/annotations/swagger-file" %}}).
