---
title: "Grammar"
weight: 40
description: "The formal ISO-14977 EBNF the parser implements, from comment preprocessing through the typed walker."
---


The formal grammar of the codescan annotation surface. This document
specifies the language a Go comment must conform to so that the
scanner classifies it, dispatches it to the right builder, and
populates the OpenAPI spec deterministically.

**Audience.** Implementers — anyone porting, extending, or debugging
the parser. Annotation authors typically need
[annotations.md]({{% relref "annotations" %}}) and [keywords.md]({{% relref "keywords" %}})
instead.

The grammar is layered:

1. **Preprocess** — comment-marker stripping (see
   [§preprocess](#preprocess)).
2. **Lex** — terminal token emission, including multi-line body
   accumulation (see [§lexer](#lexer)).
3. **Parse** — block construction, family dispatch, keyword
   classification (see [§parser](#parser)).
4. **Walk** — typed dispatch through `grammar.Walker` callbacks to
   the builders (see [§walker](#walker)).

The productions below operate on the **lexer's terminal alphabet**,
not raw text. Per-terminal lexical detail (how the lexer recognises
a number, a string, an annotation, …) is described in §lexer; the
EBNF that follows consumes pre-classified terminals.

The grammar is rigorous ISO-14977 EBNF. Required vs. optional
arguments, value typing, and family membership are
**grammar-visible** — every legality constraint expressible by token
sequencing is expressed that way.

---

## Table of contents

- [Preprocess](#preprocess)
- [Lexer](#lexer)
  - [Terminal vocabulary](#terminal-vocabulary)
  - [Body accumulation](#body-accumulation)
  - [YAML fence handling](#yaml-fence-handling)
  - [Prose classification](#prose-classification)
- [Parser](#parser)
  - [Top-level dispatch](#top-level-dispatch)
  - [Schema family](#schema-family)
  - [Operation family](#operation-family)
  - [Meta family](#meta-family)
  - [Classifier family](#classifier-family)
- [Cross-cutting productions](#cross-cutting-productions)
- [Walker](#walker)
- [Diagnostics](#diagnostics)
- [What this grammar does not describe](#what-this-grammar-does-not-describe)

---

## Preprocess

Input is a `*ast.CommentGroup` from `go/parser`. Each `*ast.Comment`
in the group is one source-level comment node (either `// …` or
`/* … */`). The preprocessor produces a flat sequence of `Line`
structs, each one source line with:

- **`Line.Text`** — content after comment-marker stripping and
  leading content-prefix trim.
- **`Line.Raw`** — content after comment-marker stripping only
  (preserves leading whitespace).
- **`Line.Pos`** — `token.Position` of the first content byte.

Stripping rules:

- For `//` comments: drop the `//` marker. `Line.Text` runs
  `trimContentPrefix` (strips leading ` \t*/|`); `Line.Raw` keeps
  the post-marker spacing.
- For `/* */` block comments: split body on newlines.
  - First line: drop the `/*` marker.
  - Continuation lines: run `stripBlockContinuation` (strips
    leading whitespace + optional `*` continuation marker + one
    following space), then `trimContentPrefix`.
  - Last line: drop the trailing `*/`.

`trimContentPrefix` strips ` \t*/` and a single trailing `|` from
the line head. It does NOT strip `-` (so YAML list markers and
markdown dash items survive intact).

For synthetic per-line comments produced by upstream tooling
(notably `parsers.ParseRoutePathAnnotation`), a `// ` prefix is
prepended before stripping so the `//` branch fires and the leading
whitespace gets shed correctly.

---

## Lexer

The lexer turns a `[]Line` into a `[]Token` ending in `TokenEOF`.
Pipeline:

1. **Line classifier** — emit one preliminary token per line
   (annotation / keyword / fence / blank / text).
2. **Body accumulator** — fold multi-line bodies (OPAQUE_YAML,
   RAW_BLOCK_*, RAW_VALUE_*) into single body tokens.
3. **Prose classifier** — re-type surviving text tokens as
   `TokenTitle` / `TokenDesc`.

### Terminal vocabulary

#### Annotation name terminals (`TokenAnnotation`)

Each recognises an annotation **name** only — positional arguments
are emitted as separate terminals.

| Terminal | Annotation |
|----------|------------|
| `ANN_MODEL` | `swagger:model` |
| `ANN_RESPONSE` | `swagger:response` |
| `ANN_PARAMETERS` | `swagger:parameters` |
| `ANN_ROUTE` | `swagger:route` |
| `ANN_OPERATION` | `swagger:operation` |
| `ANN_META` | `swagger:meta` |
| `ANN_STRFMT` | `swagger:strfmt` |
| `ANN_ALIAS` | `swagger:alias` |
| `ANN_NAME` | `swagger:name` |
| `ANN_ALLOF` | `swagger:allOf` |
| `ANN_ENUM` | `swagger:enum` |
| `ANN_IGNORE` | `swagger:ignore` |
| `ANN_DEFAULT` | `swagger:default` |
| `ANN_TYPE` | `swagger:type` |
| `ANN_ADDITIONAL_PROPERTIES` | `swagger:additionalProperties` |
| `ANN_PATTERN_PROPERTIES` | `swagger:patternProperties` |
| `ANN_FILE` | `swagger:file` |

#### Argument terminals

| Terminal | Recognises |
|----------|------------|
| `IDENT_NAME` | Identifier-shaped token. Used for every named arg and reference. |
| `JSON_VALUE` | RFC-8259 JSON literal (string / number / boolean / null / array / object). |
| `RAW_VALUE` | Verbatim non-LF text — fallback when `JSON_VALUE` recognition fails. |
| `TYPE_REF` | Closed vocab: `string` / `integer` / `number` / `boolean` / `array` / `object` / `file` / `null`. |
| `HTTP_METHOD` | `GET` / `POST` / `PUT` / `PATCH` / `HEAD` / `DELETE` / `OPTIONS` / `TRACE` (case-insensitive). |
| `URL_PATH` | RFC-3986 URL path token (used as the second positional arg of `OperationArgs`). |

#### Keyword head terminals (`TokenKeyword`)

Each recognises the keyword **name** only. See
[keywords.md]({{% relref "keywords" %}}) for the complete keyword surface.

#### Inline value terminals

The lexer types values per their lexical shape; semantic coercion
against the Go target happens in the analyzer.

| Terminal | Recognises |
|----------|------------|
| `NUMBER_VALUE` | Signed decimal literal (integer or fractional). |
| `INT_VALUE` | Unsigned decimal integer. |
| `BOOL_VALUE` | `true` / `false` (case-insensitive). |
| `STRING_VALUE` | Verbatim non-LF text. |
| `COMMA_LIST_VALUE` | Comma-separated list of strings, trim-stripped. |
| `ENUM_OPTION_VALUE` | One of a closed token set declared per keyword (`query`/`path`/… for `in:`, `csv`/`ssv`/… for `collectionFormat`). |

When the lexer fails to type a value against its keyword's expected
shape, the property reaches the analyzer with `Property.Typed.Type ==
ShapeNone` and a `CodeInvalidNumber` / `CodeInvalidInteger` /
`CodeInvalidBoolean` diagnostic is emitted.

#### Multi-line body terminals

Single tokens spanning multiple source lines. The lexer absorbs the
head and the body lines.

| Terminal | Parent keyword | Body shape |
|----------|----------------|------------|
| `RAW_BLOCK_CONSUMES` | `consumes` | Flat token list (see [sub-languages §flex-list]({{% relref "sub-languages#flex-list" %}})) |
| `RAW_BLOCK_PRODUCES` | `produces` | Flat token list |
| `RAW_BLOCK_SCHEMES` | `schemes` | Flat token list |
| `RAW_BLOCK_SECURITY` | `security` | Security requirements (see [sub-languages §security-requirements]({{% relref "sub-languages#security-requirements" %}})) |
| `RAW_BLOCK_SECURITY_DEFINITIONS` | `securityDefinitions` | YAML map |
| `RAW_BLOCK_RESPONSES` | `responses` | Response sub-language (see [sub-languages §responses]({{% relref "sub-languages#responses" %}})) |
| `RAW_BLOCK_PARAMETERS` | `parameters` | Parameter chunk sub-language (see [sub-languages §parameters]({{% relref "sub-languages#parameters" %}})) |
| `RAW_BLOCK_EXTENSIONS` | `extensions` | YAML map of `x-*` entries |
| `RAW_BLOCK_INFO_EXTENSIONS` | `infoExtensions` | YAML map of `x-*` entries |
| `RAW_BLOCK_TOS` | `tos` | Free-form prose paragraph |
| `RAW_BLOCK_EXTERNAL_DOCS` | `externalDocs` | YAML map |
| `RAW_VALUE_DEFAULT` | `default` | Raw value text |
| `RAW_VALUE_EXAMPLE` | `example` | Raw value text |
| `RAW_VALUE_ENUM` | `enum` | Comma list, JSON array, or YAML dash list |

### Body accumulation

A raw-block / raw-value keyword opens a body. The body terminates at
the **next sibling structural token** in the same family — either
another `TokenAnnotation`, another body-keyword head whose context
makes it a sibling, or `TokenEOF`.

Blank lines do NOT terminate the body. They are absorbed as visual
separators inside list-shaped bodies.

For raw-block heads, the inline post-colon value (when non-empty)
is prepended to the body as its first line. This means
`Consumes: application/json` (inline single value) and
`Consumes:\n  - application/json` (multi-line body) both yield the
same body content; consumers don't need to special-case the inline
form.

### YAML fence handling

A line whose trimmed content is exactly `---` opens (or closes) a
YAML fence. While the cursor sits between matching fences:

- Annotation and keyword recognition is suspended; every line emits
  as `tokenRawLine` carrying the verbatim source text.
- The body accumulator captures the fenced region as a single
  `OPAQUE_YAML` token attached to the surrounding annotation
  (typically `swagger:operation` or a fenced extensions body).
- A missing closing fence emits a `CodeUnterminatedFence`
  diagnostic; the `OPAQUE_YAML` token is marked truncated and the
  builder degrades gracefully.

### Prose classification

Surviving `tokenText` tokens (not consumed by a body, not an
annotation or keyword head) re-type as either `TokenTitle` or
`TokenDesc` per three heuristics evaluated in order:

1. **Blank-line split** — a blank line inside the prose run ends
   the title and starts the description.
2. **Closing punctuation** — if the first prose line ends with
   Unicode punctuation, the title is just that one line.
3. **Markdown ATX heading** — if the first prose line matches
   markdown's `# Heading` shape, the `#` markers are stripped and
   the line becomes the title.

When no heuristic fires, the entire prose run is title.

See [sub-languages.md §prose-classification]({{% relref "sub-languages#prose-classification" %}})
for the author-facing description.

---

## Parser

The parser consumes the lexer's terminal stream and produces typed
`Block` values, one per `*ast.CommentGroup`. A single comment group
may produce MORE than one Block when multiple annotations appear
(each annotation closes the preceding Block and opens a fresh one).

### Top-level dispatch

```ebnf
CommentBlock     = AnnotatedBlock | UnboundBlock ;

AnnotatedBlock   = SchemaBlock
                 | OperationFamilyBlock
                 | MetaBlock
                 | ClassifierBlock ;

UnboundBlock     = [ Description ] , UnboundBlockBody ;
```

{{< railroad >}}
AnnotatedBlock       = SchemaBlock | OperationFamilyBlock | MetaBlock | ClassifierBlock ;
SchemaBlock          = SchemaAnnotation , [ Title ] , [ Description ] , SchemaAnnotationBody ;
OperationFamilyBlock = RouteBlock | InlineOperationBlock ;
MetaBlock            = ANN_META , [ Title ] , [ Description ] , MetaBody ;
ClassifierBlock      = StrfmtBlock | AliasBlock | AllOfBlock | EnumBlock | IgnoreBlock | DefaultClassifierBlock | TypeBlock | FileBlock ;
{{< /railroad >}}

The dispatcher reads the first `ANN_*` terminal; its identity
selects the family. If no annotation appears, the input is an
`UnboundBlock` — typically a Go struct field with description-only
documentation.

`Block.AnnotationKind()` returns the family discriminator.
`Block.AnnotationArg()` returns the leading IDENT argument (if any)
without requiring the caller to type-assert on the typed Block
kind.

### Schema family

Bodies of `swagger:model`, `swagger:parameters`, `swagger:response`,
`swagger:name`.

```ebnf
SchemaBlock          = SchemaAnnotation
                     , [ Title ]
                     , [ Description ]
                     , SchemaAnnotationBody ;

SchemaAnnotation     = ModelAnnotation
                     | ResponseAnnotation
                     | ParametersAnnotation
                     | NameAnnotation ;

ModelAnnotation      = ANN_MODEL ,      [ IDENT_NAME ] ;
ResponseAnnotation   = ANN_RESPONSE ,   [ IDENT_NAME ] ;
ParametersAnnotation = ANN_PARAMETERS , IDENT_NAME , { IDENT_NAME } ;
NameAnnotation       = ANN_NAME ,       IDENT_NAME ;

SchemaAnnotationBody = { SchemaBodyItem } ;
UnboundBlockBody     = { SchemaBodyItem } ;

SchemaBodyItem       = Validation
                     | SchemaDecorator
                     | ExtensionsBlock
                     | ExternalDocsBlock
                     | BLANK ;

Validation           = NumericValidation
                     | StringValidation
                     | ArrayValidation
                     | EnumValidation
                     | RequiredLine
                     | ReadOnlyLine ;

NumericValidation    = NumericKw , NUMBER_VALUE ;
NumericKw            = KW_MAXIMUM | KW_MINIMUM | KW_MULTIPLE_OF ;

StringValidation     = KW_PATTERN , STRING_VALUE
                     | StringLengthKw , INT_VALUE ;
StringLengthKw       = KW_MAX_LENGTH | KW_MIN_LENGTH ;

ArrayValidation      = ArrayCountKw , INT_VALUE
                     | KW_UNIQUE , BOOL_VALUE
                     | KW_COLLECTION_FORMAT , ENUM_OPTION_VALUE ;
ArrayCountKw         = KW_MAX_ITEMS | KW_MIN_ITEMS ;

EnumValidation       = RAW_VALUE_ENUM ;
RequiredLine         = KW_REQUIRED , BOOL_VALUE ;
ReadOnlyLine         = KW_READ_ONLY , BOOL_VALUE ;

SchemaDecorator      = RAW_VALUE_DEFAULT
                     | RAW_VALUE_EXAMPLE
                     | DiscriminatorLine
                     | DeprecatedLine ;

DiscriminatorLine    = KW_DISCRIMINATOR , BOOL_VALUE ;
DeprecatedLine       = KW_DEPRECATED , BOOL_VALUE ;
```

{{< railroad >}}
SchemaBodyItem    = Validation | SchemaDecorator | ExtensionsBlock | ExternalDocsBlock | BLANK ;
Validation        = NumericValidation | StringValidation | ArrayValidation | EnumValidation | RequiredLine | ReadOnlyLine ;
NumericValidation = ( KW_MAXIMUM | KW_MINIMUM | KW_MULTIPLE_OF ) , NUMBER_VALUE ;
StringValidation  = ( KW_PATTERN , STRING_VALUE ) | ( ( KW_MAX_LENGTH | KW_MIN_LENGTH ) , INT_VALUE ) ;
ArrayValidation   = ( ( KW_MAX_ITEMS | KW_MIN_ITEMS ) , INT_VALUE ) | ( KW_UNIQUE , BOOL_VALUE ) | ( KW_COLLECTION_FORMAT , ENUM_OPTION_VALUE ) ;
EnumValidation    = RAW_VALUE_ENUM ;
RequiredLine      = KW_REQUIRED , BOOL_VALUE ;
ReadOnlyLine      = KW_READ_ONLY , BOOL_VALUE ;
SchemaDecorator   = RAW_VALUE_DEFAULT | RAW_VALUE_EXAMPLE | ( KW_DISCRIMINATOR , BOOL_VALUE ) | ( KW_DEPRECATED , BOOL_VALUE ) ;
{{< /railroad >}}

### Operation family

`swagger:route` and `swagger:operation` are distinct block productions
because their bodies differ structurally — `swagger:route` accepts
the structured keyword surface; `swagger:operation` accepts an
`OPAQUE_YAML` body.

```ebnf
OperationFamilyBlock = RouteBlock | InlineOperationBlock ;

RouteBlock           = ANN_ROUTE , OperationArgs
                     , [ Title ]
                     , [ Description ]
                     , RouteBody ;

InlineOperationBlock = ANN_OPERATION , OperationArgs
                     , [ Title ]
                     , [ Description ]
                     , InlineOperationBody ;

OperationArgs        = HTTP_METHOD , URL_PATH , { IDENT_NAME } , IDENT_NAME ;
                      (* Trailing IDENT_NAME is the OperationID;
                         the run between URL_PATH and the OpID is
                         the tag list. *)

RouteBody            = { CommonOperationBodyItem | BLANK } ;

InlineOperationBody  = { CommonOperationBodyItem
                       | OPAQUE_YAML
                       | BLANK } ;

CommonOperationBodyItem = OperationKeyword
                        | OperationDecorator
                        | OperationRawBlock
                        | ExtensionsBlock
                        | ExternalDocsBlock ;

OperationKeyword     = KW_SCHEMES , COMMA_LIST_VALUE ;

OperationDecorator   = DeprecatedLine ;

OperationRawBlock    = RAW_BLOCK_CONSUMES
                     | RAW_BLOCK_PRODUCES
                     | RAW_BLOCK_SECURITY
                     | RAW_BLOCK_RESPONSES
                     | RAW_BLOCK_PARAMETERS ;
```

{{< railroad >}}
RouteBlock = ANN_ROUTE , OperationArgs , [ Title ] , [ Description ] , RouteBody ;
{{< /railroad >}}

{{< railroad >}}
InlineOperationBlock = ANN_OPERATION , OperationArgs , [ Title ] , [ Description ] , InlineOperationBody ;
{{< /railroad >}}

_…where both share the header arguments:_

{{< railroad >}}
OperationArgs = HTTP_METHOD , URL_PATH , { IDENT_NAME } , IDENT_NAME ;
{{< /railroad >}}

The `<GoIdent> swagger:route ...` godoc-prefix exception (which
allows a leading Go identifier on the route annotation line) is
absorbed by the lexer; the EBNF sees a plain `ANN_ROUTE`.

### Meta family

`swagger:meta` defines top-of-spec metadata.

```ebnf
MetaBlock            = ANN_META
                     , [ Title ]
                     , [ Description ]
                     , MetaBody ;

MetaBody             = { MetaBodyItem | BLANK } ;

MetaBodyItem         = MetaKeyword
                     | MetaRawBlock
                     | ExtensionsBlock
                     | InfoExtensionsBlock
                     | ExternalDocsBlock ;

MetaKeyword          = KW_VERSION , STRING_VALUE
                     | KW_HOST , STRING_VALUE
                     | KW_BASE_PATH , STRING_VALUE
                     | KW_LICENSE , STRING_VALUE
                     | KW_CONTACT , STRING_VALUE
                     | KW_SCHEMES , COMMA_LIST_VALUE ;

MetaRawBlock         = RAW_BLOCK_CONSUMES
                     | RAW_BLOCK_PRODUCES
                     | RAW_BLOCK_SCHEMES
                     | RAW_BLOCK_SECURITY
                     | RAW_BLOCK_SECURITY_DEFINITIONS
                     | RAW_BLOCK_TOS ;
```

{{< railroad >}}
MetaBlock    = ANN_META , [ Title ] , [ Description ] , MetaBody ;
MetaBodyItem = MetaKeyword | MetaRawBlock | ExtensionsBlock | InfoExtensionsBlock | ExternalDocsBlock ;
MetaKeyword  = ( ( KW_VERSION | KW_HOST | KW_BASE_PATH | KW_LICENSE | KW_CONTACT ) , STRING_VALUE ) | ( KW_SCHEMES , COMMA_LIST_VALUE ) ;
MetaRawBlock = RAW_BLOCK_CONSUMES | RAW_BLOCK_PRODUCES | RAW_BLOCK_SCHEMES | RAW_BLOCK_SECURITY | RAW_BLOCK_SECURITY_DEFINITIONS | RAW_BLOCK_TOS ;
{{< /railroad >}}

### Classifier family

Single-purpose annotations that classify the surrounding declaration
without carrying their own body.

```ebnf
ClassifierBlock      = StrfmtBlock
                     | AliasBlock
                     | AllOfBlock
                     | EnumBlock
                     | IgnoreBlock
                     | DefaultClassifierBlock
                     | TypeBlock
                     | FileBlock ;

StrfmtBlock          = ANN_STRFMT , IDENT_NAME , [ Title ] , [ Description ] ;
AliasBlock           = ANN_ALIAS ,  [ IDENT_NAME ] , [ Title ] , [ Description ] ;
AllOfBlock           = ANN_ALLOF , [ Title ] , [ Description ] ;
EnumBlock            = ANN_ENUM , [ IDENT_NAME ] , [ Title ] , [ Description ] ;
IgnoreBlock          = ANN_IGNORE , [ Title ] , [ Description ] ;
DefaultClassifierBlock = ANN_DEFAULT , [ Title ] , [ Description ] ;
TypeBlock            = ANN_TYPE , TYPE_REF , [ Title ] , [ Description ] ;
FileBlock            = ANN_FILE , [ Title ] , [ Description ] ;
```

{{< railroad >}}
ClassifierBlock        = StrfmtBlock | AliasBlock | AllOfBlock | EnumBlock | IgnoreBlock | DefaultClassifierBlock | TypeBlock | FileBlock ;
StrfmtBlock            = ANN_STRFMT , IDENT_NAME , [ Title ] , [ Description ] ;
AliasBlock             = ANN_ALIAS , [ IDENT_NAME ] , [ Title ] , [ Description ] ;
AllOfBlock             = ANN_ALLOF , [ Title ] , [ Description ] ;
EnumBlock              = ANN_ENUM , [ IDENT_NAME ] , [ Title ] , [ Description ] ;
IgnoreBlock            = ANN_IGNORE , [ Title ] , [ Description ] ;
DefaultClassifierBlock = ANN_DEFAULT , [ Title ] , [ Description ] ;
TypeBlock              = ANN_TYPE , TYPE_REF , [ Title ] , [ Description ] ;
FileBlock              = ANN_FILE , [ Title ] , [ Description ] ;
{{< /railroad >}}

Classifiers are stateless markers — they carry no validation body
of their own. The surrounding declaration's other annotations (or
the absence thereof) determine where the classification lands.

---

## Cross-cutting productions

These appear in multiple families and share a single production.

```ebnf
ExtensionsBlock      = RAW_BLOCK_EXTENSIONS ;
InfoExtensionsBlock  = RAW_BLOCK_INFO_EXTENSIONS ;
ExternalDocsBlock    = RAW_BLOCK_EXTERNAL_DOCS ;

Title                = TokenTitle ;
Description          = TokenDesc , { TokenDesc | BLANK , TokenDesc } ;
BLANK                = TokenBlank ;
```

Vendor extensions (`ExtensionsBlock`, `InfoExtensionsBlock`) accept
YAML map bodies; non-`x-*` keys emit `CodeInvalidAnnotation` and
drop. The lexer additionally surfaces them via
`Block.Extensions()` with an `Extension.Source` discriminator
(`KwExtensions` vs `KwInfoExtensions`) so consumers can route to
the correct spec field (`spec.extensions` vs `info.extensions`).

---

## Walker

`Block.Walk(grammar.Walker{...})` dispatches Properties through
typed callbacks. The Walker maps a Property to a callback by
`Keyword.Shape`:

| Shape | Callback | Payload |
|-------|----------|---------|
| `ShapeNumber` | `Number` | `(p, float64, exclusive bool)` |
| `ShapeInt` | `Integer` | `(p, int64)` |
| `ShapeBool` | `Bool` | `(p, bool)` |
| `ShapeString` | `String` | `(p, string)` — value on `p.Value` |
| `ShapeEnumOption` | `String` | `(p, string)` — closed-vocab token on `p.Typed.String` |
| `ShapeRawBlock` | `Raw` | `(p)` — caller reads `p.Body` / `p.Raw` |
| `ShapeRawValue` | `Raw` | `(p)` |
| `ShapeCommaList` | `Raw` | `(p)` — caller splits via `Property.AsList` |
| `ShapeNone` (failed typing) | `Raw` | `(p)` — diagnostic fired separately |

Additional callbacks fire outside the per-Property dispatch:

- `Title(s string)` — once, before any property, if non-empty.
- `Description(s string)` — once, before any property, if non-empty.
- `Extension(ext grammar.Extension)` — once per typed extension.
- `Diagnostic(d grammar.Diagnostic)` — block-level diagnostics fire
  before Title; per-property diagnostics fire immediately before
  the property's main callback.

`Walker.FilterDepth` gates property callbacks by `Property.ItemsDepth`.
Pass `0` for level-0 properties (default); pass `N` for items-level
N; pass `AllDepths` (-1) for every depth.

For full Walker contract see the
[`grammar` package README](../internal/parsers/grammar/README.md#walker-contract).

---

## Diagnostics

The grammar emits typed diagnostics for malformed input, recovered
where possible:

| Code | Severity | Trigger |
|------|----------|---------|
| `CodeInvalidAnnotation` | Warning | Unknown tag, malformed annotation arg, dropped malformed property |
| `CodeInvalidNumber` | Warning | Number-typed value failed lexical parse |
| `CodeInvalidInteger` | Warning | Integer-typed value failed lexical parse |
| `CodeInvalidBoolean` | Warning | Boolean-typed value failed lexical parse |
| `CodeShapeMismatch` | Warning | Keyword applied to a schema type that doesn't accept it (e.g. `minLength` on a number) |
| `CodeContextInvalid` | Warning | Keyword used outside its legal annotation context |
| `CodeUnsupportedInSimpleSchema` | Warning | Full-schema-only keyword used in SimpleSchema (non-body param, header) |
| `CodeInvalidYAMLExtensions` | Warning | YAML parse failed inside an extensions body |
| `CodeUnterminatedFence` | Warning | YAML fence opened but not closed before EOF |

All diagnostics drop the offending property / annotation /
extension and continue the build. The accumulator on
`common.Builder` collects them in source order; the consumer's
`OnDiagnostic` callback (if wired) fires inline.

---

## What this grammar does not describe

The grammar's job ends at producing typed `Property` and `Block`
values. The analyzer (builders / spec orchestrator) owns:

- **Type coercion** — `default: 1.5` against an `integer` schema is
  a lexical success and an analyzer rejection. `validations.CoerceValue`
  and `validations.ParseDefault` apply the schema-type-aware
  coercion at write time.
- **Cross-reference resolution** — `$ref` targets, alias-chain
  resolution, post-decl discovery. The grammar emits the names; the
  analyzer resolves them.
- **Schema-shape gating** — `validations.IsLegalForType` decides
  whether `minLength` applies to the resolved schema type. The
  grammar always emits the property; the handler dispatch decides
  whether to write it.
- **Ordering & merging across multiple comment groups** — when
  several `swagger:parameters Foo Bar Baz` declarations contribute
  to the same operation, the spec builder merges them.

The grammar is also deliberately **single-pass** — it never
revisits a `*ast.CommentGroup` after `Parse(cg)` returns. The
`common.Builder` blockCache memoises results across the analyzer's
recursive type descent (see
[`common` README §blockcache](../internal/builders/common/README.md#blockcache)).
