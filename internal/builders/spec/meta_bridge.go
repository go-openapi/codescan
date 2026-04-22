// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/parsers/helpers"
	"github.com/go-openapi/codescan/internal/scanner/classify"
	"github.com/go-openapi/loads/fmts"
	"github.com/go-openapi/spec"
	yaml "go.yaml.in/yaml/v3"
)

// rxStripTitleComments mirrors the legacy regex used in NewMetaParser's
// setTitle callback. A meta title of the form
// `Package <identifier> <rest>` has the leading Go package marker
// stripped so the emitted Info.Title is just `<rest>`.
var rxStripTitleComments = regexp.MustCompile(`^[^\p{L}]*[Pp]ackage\p{Zs}+[^\p{Zs}]+\p{Zs}*`)

// applyMetaBlock parses the meta comment group via the grammar
// parser and dispatches each level-0 property into the matching
// *spec.Swagger field. Replaces parsers.NewMetaParser + SectionedParser
// with no behavior change: title/description come from the grammar's
// prose split (blank-line or punctuation/markdown heuristic, same as
// the legacy CollectScannerTitleDescription helper), and each
// top-level keyword's body is handed to the setter that v1 wired
// behind the scenes.
//
// swspec may have a nil Info field on entry; the helper allocates
// one before writing the first Info.* value.
func applyMetaBlock(swspec *spec.Swagger, block grammar.Block) error {
	if swspec.Info == nil {
		swspec.Info = new(spec.Info)
	}
	title, desc := helpers.CollectScannerTitleDescription(block.ProseLines())
	joinedTitle := helpers.JoinDropLast(title)
	if joinedTitle != "" {
		joinedTitle = rxStripTitleComments.ReplaceAllString(joinedTitle, "")
	}
	swspec.Info.Title = joinedTitle
	swspec.Info.Description = helpers.JoinDropLast(desc)

	for p := range block.Properties() {
		if p.ItemsDepth != 0 {
			continue
		}
		if err := dispatchMetaKeyword(p, swspec); err != nil {
			return err
		}
	}
	return nil
}

func dispatchMetaKeyword(p grammar.Property, swspec *spec.Swagger) error {
	if dispatchMetaSimple(p, swspec) {
		return nil
	}
	return dispatchMetaYAMLBlock(p, swspec)
}

// dispatchMetaSimple handles the synchronous, non-YAML keywords
// whose body dispatch cannot fail.
func dispatchMetaSimple(p grammar.Property, swspec *spec.Swagger) bool {
	switch p.Keyword.Name {
	case "tos":
		swspec.Info.TermsOfService = helpers.JoinDropLast(helpers.DropEmpty(p.Body))
	case "consumes":
		swspec.Consumes = helpers.YAMLListBody(p.Body)
	case "produces":
		swspec.Produces = helpers.YAMLListBody(p.Body)
	case "schemes":
		swspec.Schemes = helpers.SchemesList(p.Value)
	case "security":
		swspec.Security = helpers.SecurityRequirements(p.Body)
	case "version":
		swspec.Info.Version = strings.TrimSpace(p.Value)
	case "host":
		host := strings.TrimSpace(p.Value)
		if host == "" {
			host = "localhost"
		}
		swspec.Host = host
	case "basePath":
		swspec.BasePath = strings.TrimSpace(p.Value)
	case "license":
		swspec.Info.License = parseLicense(strings.TrimSpace(p.Value))
	default:
		return false
	}
	return true
}

// dispatchMetaYAMLBlock handles the keywords that can fail:
// securityDefinitions, infoExtensions, extensions, contact.
func dispatchMetaYAMLBlock(p grammar.Property, swspec *spec.Swagger) error {
	switch p.Keyword.Name {
	case "contact":
		contact, err := parseContactInfo(strings.TrimSpace(p.Value))
		if err != nil {
			return err
		}
		swspec.Info.Contact = contact
	case "securityDefinitions":
		return unmarshalYAMLBody(p.Body, func(data []byte) error {
			var d spec.SecurityDefinitions
			if err := json.Unmarshal(data, &d); err != nil {
				return err
			}
			swspec.SecurityDefinitions = d
			return nil
		})
	case "infoExtensions":
		return unmarshalYAMLBody(p.Body, func(data []byte) error {
			return applyInfoExtensions(data, swspec)
		})
	case "extensions":
		return unmarshalYAMLBody(p.Body, func(data []byte) error {
			return applyMetaExtensions(data, swspec)
		})
	}
	return nil
}

func applyInfoExtensions(data []byte, swspec *spec.Swagger) error {
	var d spec.Extensions
	if err := json.Unmarshal(data, &d); err != nil {
		return err
	}
	if err := validateExtensionNames(d); err != nil {
		return err
	}
	swspec.Info.Extensions = d
	return nil
}

func applyMetaExtensions(data []byte, swspec *spec.Swagger) error {
	var d spec.Extensions
	if err := json.Unmarshal(data, &d); err != nil {
		return err
	}
	if err := validateExtensionNames(d); err != nil {
		return err
	}
	swspec.Extensions = d
	return nil
}

// unmarshalYAMLBody mirrors parsers.YAMLParser.Parse: the block
// body (`---` fence contents, preserving indent) is yaml-unmarshal'd,
// converted to JSON via fmts.YAMLToJSON, and handed to the setter.
func unmarshalYAMLBody(body []string, setter func([]byte) error) error {
	cleaned := removeYAMLIndent(body)
	if len(cleaned) == 0 {
		return nil
	}
	yamlContent := strings.Join(cleaned, "\n")
	var v any
	if err := yaml.Unmarshal([]byte(yamlContent), &v); err != nil {
		return err
	}
	raw, err := fmts.YAMLToJSON(v)
	if err != nil {
		return err
	}
	data, err := raw.MarshalJSON()
	if err != nil {
		return err
	}
	return setter(data)
}

// removeYAMLIndent mirrors parsers.removeYamlIndent — strip the
// common leading-indent detected on the first non-empty line.
func removeYAMLIndent(body []string) []string {
	cleaned := helpers.DropEmpty(body)
	if len(cleaned) == 0 {
		return nil
	}
	indent := leadingWhitespaceLen(cleaned[0])
	if indent == 0 {
		return cleaned
	}
	out := make([]string, 0, len(cleaned))
	for _, line := range cleaned {
		if len(line) >= indent {
			out = append(out, line[indent:])
		} else {
			out = append(out, line)
		}
	}
	return out
}

func leadingWhitespaceLen(s string) int {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i
}

// ErrBadExtensionName is the sentinel used when a meta extension key
// does not start with `x-` or `X-`. Mirrors the legacy behavior of
// metaVendorExtensibleSetter's reject-with-error path.
var ErrBadExtensionName = errors.New("invalid schema extension name, should start from `x-`")

// validateExtensionNames mirrors the legacy rxAllowedExtensions
// check — every vendor extension key must begin with `x-` or `X-`.
func validateExtensionNames(ext spec.Extensions) error {
	for k := range ext {
		if !classify.IsAllowedExtension(k) {
			return fmt.Errorf("%w: %s", ErrBadExtensionName, k)
		}
	}
	return nil
}

// parseContactInfo parses a `Name <email> URL` shaped contact line.
func parseContactInfo(line string) (*spec.ContactInfo, error) {
	nameEmail, url := splitURL(line)
	var name, email string
	if nameEmail != "" {
		addr, err := mail.ParseAddress(nameEmail)
		if err != nil {
			return nil, err
		}
		name, email = addr.Name, addr.Address
	}
	return &spec.ContactInfo{
		ContactInfoProps: spec.ContactInfoProps{
			URL:   url,
			Name:  name,
			Email: email,
		},
	}, nil
}

func parseLicense(line string) *spec.License {
	name, url := splitURL(line)
	return &spec.License{
		LicenseProps: spec.LicenseProps{
			Name: name,
			URL:  url,
		},
	}
}

var httpFTPScheme = regexp.MustCompile(`(?:(?:ht|f)tp|ws)s?://`)

func splitURL(line string) (notURL, url string) {
	str := strings.TrimSpace(line)
	parts := httpFTPScheme.FindStringIndex(str)
	if len(parts) == 0 {
		if str != "" {
			notURL = str
		}
		return notURL, ""
	}
	notURL = strings.TrimSpace(str[:parts[0]])
	url = strings.TrimSpace(str[parts[0]:])
	return notURL, url
}
