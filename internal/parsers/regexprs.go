// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parsers

import (
	"fmt"
	"regexp"
)

const (
	// rxCommentPrefix matches the leading comment noise that precedes an
	// annotation keyword on a raw comment line: whitespace, tabs, slashes,
	// asterisks, dashes, optional markdown table pipe, then any trailing
	// spaces. Mirrors the prefix class used by rxUncommentHeaders so
	// Matches() can still see through the `//` / `*` / ` * ` comment
	// prefixes on raw lines.
	//
	// Annotations must START the comment line — any prose before the
	// swagger:xxx keyword disqualifies the line: an annotation buried in prose is ignored.
	//
	// Example:
	// `swagger:strfmt` buried inside the sentence
	// `// MAC is a text-marshalable ... swagger:strfmt so ...` is ignored and no longer captures
	// "so" instead of the intended strfmt name.
	//
	// The sole documented-by-example exception is `swagger:route`, which is
	// allowed to follow a single godoc identifier (see rxRoutePrefix).
	rxCommentPrefix = `^[\p{Zs}\t/\*-]*\|?\p{Zs}*`

	// rxRoutePrefix extends rxCommentPrefix with an OPTIONAL single leading
	// identifier. Godoc convention places the function/type name before the
	// annotation body, e.g. `// DoBad swagger:route GET /path`. Without
	// this allowance we would reject every `swagger:route` annotation
	// attached to a documented handler. The allowance is intentionally
	// narrow — ONE identifier, then whitespace — so multi-word prose
	// prefixes still fail.
	//
	// This exception is reserved for `swagger:route`. All other annotations
	// must start the comment line, per rxCommentPrefix.
	rxRoutePrefix = rxCommentPrefix + `(?:\p{L}[\p{L}\p{N}\p{Pd}\p{Pc}]*\p{Zs}+)?`

	rxMethod = "(\\p{L}+)"
	rxPath   = "((?:/[\\p{L}\\p{N}\\p{Pd}\\p{Pc}{}\\-\\.\\?_~%!$&'()*+,;=:@/]*)+/?)"
	rxOpTags = "(\\p{L}[\\p{L}\\p{N}\\p{Pd}\\.\\p{Pc}\\p{Zs}]+)"
	rxOpID   = "((?:\\p{L}[\\p{L}\\p{N}\\p{Pd}\\p{Pc}]+)+)"

	rxMaximumFmt    = rxCommentPrefix + "%s[Mm]ax(?:imum)?\\p{Zs}*:\\p{Zs}*([\\<=])?\\p{Zs}*([\\+-]?(?:\\p{N}+\\.)?\\p{N}+)(?:\\.)?$"
	rxMinimumFmt    = rxCommentPrefix + "%s[Mm]in(?:imum)?\\p{Zs}*:\\p{Zs}*([\\>=])?\\p{Zs}*([\\+-]?(?:\\p{N}+\\.)?\\p{N}+)(?:\\.)?$"
	rxMultipleOfFmt = rxCommentPrefix + "%s[Mm]ultiple\\p{Zs}*[Oo]f\\p{Zs}*:\\p{Zs}*([\\+-]?(?:\\p{N}+\\.)?\\p{N}+)(?:\\.)?$"

	rxMaxLengthFmt        = rxCommentPrefix + "%s[Mm]ax(?:imum)?(?:\\p{Zs}*[\\p{Pd}\\p{Pc}]?[Ll]en(?:gth)?)\\p{Zs}*:\\p{Zs}*(\\p{N}+)(?:\\.)?$"
	rxMinLengthFmt        = rxCommentPrefix + "%s[Mm]in(?:imum)?(?:\\p{Zs}*[\\p{Pd}\\p{Pc}]?[Ll]en(?:gth)?)\\p{Zs}*:\\p{Zs}*(\\p{N}+)(?:\\.)?$"
	rxPatternFmt          = rxCommentPrefix + "%s[Pp]attern\\p{Zs}*:\\p{Zs}*(.*)$"
	rxCollectionFormatFmt = rxCommentPrefix + "%s[Cc]ollection(?:\\p{Zs}*[\\p{Pd}\\p{Pc}]?[Ff]ormat)\\p{Zs}*:\\p{Zs}*(.*)$"
	rxEnumFmt             = rxCommentPrefix + "%s[Ee]num\\p{Zs}*:\\p{Zs}*(.*)$"
	rxDefaultFmt          = rxCommentPrefix + "%s[Dd]efault\\p{Zs}*:\\p{Zs}*(.*)$"
	rxExampleFmt          = rxCommentPrefix + "%s[Ee]xample\\p{Zs}*:\\p{Zs}*(.*)$"

	rxMaxItemsFmt = rxCommentPrefix + "%s[Mm]ax(?:imum)?(?:\\p{Zs}*|[\\p{Pd}\\p{Pc}]|\\.)?[Ii]tems\\p{Zs}*:\\p{Zs}*(\\p{N}+)(?:\\.)?$"
	rxMinItemsFmt = rxCommentPrefix + "%s[Mm]in(?:imum)?(?:\\p{Zs}*|[\\p{Pd}\\p{Pc}]|\\.)?[Ii]tems\\p{Zs}*:\\p{Zs}*(\\p{N}+)(?:\\.)?$"
	rxUniqueFmt   = rxCommentPrefix + "%s[Uu]nique\\p{Zs}*:\\p{Zs}*(true|false)(?:\\.)?$"

	rxItemsPrefixFmt = "(?:[Ii]tems[\\.\\p{Zs}]*){%d}"
)

var (
	// rxSwaggerAnnotation matches `swagger:<name>` anywhere on a comment
	// line where it is preceded by whitespace, `/`, or the start of the
	// line. Kept loose because it is the classification regex consumed
	// by scanner.index.ExtractAnnotation (and parsers.HasAnnotation),
	// where `swagger:route` is allowed to follow a godoc-style
	// identifier (e.g. `// MyHandler swagger:route GET /path`) per
	// rxRoutePrefix.
	//
	// Do NOT use this regex as a block terminator — it triggers on
	// mid-prose mentions like `// carries swagger:ignore, so ...` and
	// truncates descriptions. Use rxSwaggerAnnotationStrict for that.
	rxSwaggerAnnotation = regexp.MustCompile(`(?:^|[\s/])swagger:([\p{L}\p{N}\p{Pd}\p{Pc}]+)`)

	rxFileUpload         = regexp.MustCompile(rxCommentPrefix + `swagger:file`)
	rxStrFmt             = regexp.MustCompile(rxCommentPrefix + `swagger:strfmt\p{Zs}*(\p{L}[\p{L}\p{N}\p{Pd}\p{Pc}]+)(?:\.)?$`)
	rxAlias              = regexp.MustCompile(rxCommentPrefix + `swagger:alias`)
	rxName               = regexp.MustCompile(rxCommentPrefix + `swagger:name\p{Zs}*(\p{L}[\p{L}\p{N}\p{Pd}\p{Pc}\.]+)(?:\.)?$`)
	rxAllOf              = regexp.MustCompile(rxCommentPrefix + `swagger:allOf\p{Zs}*(\p{L}[\p{L}\p{N}\p{Pd}\p{Pc}\.]+)?(?:\.)?$`)
	rxModelOverride      = regexp.MustCompile(rxCommentPrefix + `swagger:model\p{Zs}*(\p{L}[\p{L}\p{N}\p{Pd}\p{Pc}]+)?(?:\.)?$`)
	rxResponseOverride   = regexp.MustCompile(rxCommentPrefix + `swagger:response\p{Zs}*(\p{L}[\p{L}\p{N}\p{Pd}\p{Pc}]+)?(?:\.)?$`)
	rxParametersOverride = regexp.MustCompile(rxCommentPrefix + `swagger:parameters\p{Zs}*(\p{L}[\p{L}\p{N}\p{Pd}\p{Pc}\p{Zs}]+)(?:\.)?$`)
	rxEnum               = regexp.MustCompile(rxCommentPrefix + `swagger:enum\p{Zs}*(\p{L}[\p{L}\p{N}\p{Pd}\p{Pc}]+)(?:\.)?$`)
	rxIgnoreOverride     = regexp.MustCompile(rxCommentPrefix + `swagger:ignore\p{Zs}*(\p{L}[\p{L}\p{N}\p{Pd}\p{Pc}]+)?(?:\.)?$`)
	rxDefault            = regexp.MustCompile(rxCommentPrefix + `swagger:default\p{Zs}*(\p{L}[\p{L}\p{N}\p{Pd}\p{Pc}]+)(?:\.)?$`)
	rxType               = regexp.MustCompile(rxCommentPrefix + `swagger:type\p{Zs}*(\p{L}[\p{L}\p{N}\p{Pd}\p{Pc}]+)(?:\.)?$`)
	rxRoute              = regexp.MustCompile(
		rxRoutePrefix +
			"swagger:route\\p{Zs}*" +
			rxMethod +
			"\\p{Zs}*" +
			rxPath +
			"(?:\\p{Zs}+" +
			rxOpTags +
			")?\\p{Zs}+" +
			rxOpID + "\\p{Zs}*$")
	rxUncommentHeaders = regexp.MustCompile(`^[\p{Zs}\t/\*-]*\|?`)
	rxOperation        = regexp.MustCompile(
		rxCommentPrefix +
			"swagger:operation\\p{Zs}*" +
			rxMethod +
			"\\p{Zs}*" +
			rxPath +
			"(?:\\p{Zs}+" +
			rxOpTags +
			")?\\p{Zs}+" +
			rxOpID + "\\p{Zs}*$")

	rxIndent            = regexp.MustCompile(`[\p{Zs}\t]*/*[\p{Zs}\t]*[^\p{Zs}\t]`)
	rxNotIndent         = regexp.MustCompile(`[^\p{Zs}\t]`)
	rxPunctuationEnd    = regexp.MustCompile(`\p{Po}$`)
	rxTitleStart        = regexp.MustCompile(`^[#]+\p{Zs}+`)
	rxAllowedExtensions = regexp.MustCompile(`^[Xx]-`)

	rxIn         = regexp.MustCompile(rxCommentPrefix + `[Ii]n\p{Zs}*:\p{Zs}*(query|path|header|body|formData)(?:\.)?$`)
	rxRequired   = regexp.MustCompile(rxCommentPrefix + `[Rr]equired\p{Zs}*:\p{Zs}*(true|false)(?:\.)?$`)
	rxSecurity   = regexp.MustCompile(rxCommentPrefix + `[Ss]ecurity\p{Zs}*[Dd]efinitions:`)
	rxResponses  = regexp.MustCompile(rxCommentPrefix + `[Rr]esponses\p{Zs}*:`)
	rxParameters = regexp.MustCompile(rxCommentPrefix + `[Pp]arameters\p{Zs}*:`)
	rxExtensions = regexp.MustCompile(rxCommentPrefix + `[Ee]xtensions\p{Zs}*:`)
)

func Rxf(rxp, ar string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(rxp, ar))
}
