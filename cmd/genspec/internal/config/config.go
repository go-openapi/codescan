// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/x/term"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec/internal/diagnostics"
	"github.com/go-openapi/codescan/cmd/genspec/internal/sentinel"
	"github.com/go-openapi/codescan/cmd/internal/cliconf"
	"github.com/go-openapi/codescan/cmd/internal/cliopts"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/swag/conv"
)

// What -color can be told. "auto" is the only one that looks at where it is writing.
const (
	// ColorAuto enables colorized diagnostic output on a tty only.
	ColorAuto = "auto"

	// ColorAlways colorized diagnostic output without checking for a terminal.
	ColorAlways = "always"

	// ColorNever disables colorized diagnostic output.
	ColorNever = "never"
)

// What -fail-on can be told: the least serious diagnostic that still makes the command exit
// non-zero.
//
// "never" is the default because a scan that emits warnings is the ordinary case - a codebase with
// nothing to say about it is the exception - and a command that failed the build over one would
// mostly teach people to stop reading it.
const (
	FailNever   = "never"
	FailError   = "error"
	FailWarning = "warning"
)

// What -format can be told.
//
// "auto" reads the extension of -output, or defaults to JSON,
// which is what a document going to standard output does.
const (
	FormatAuto = "auto"
	FormatJSON = "json"
	FormatYAML = "yaml"
)

// Config is the whole command line: the library's own knobs, plus this command's.
type Config struct {
	set *flag.FlagSet

	// scan is every knob the library takes, declared in internal/cliopts and shared with the other
	// commands, so that a flag means the same thing whichever one you reach for.
	scan *cliopts.Values

	// configFile is which file is read before the flags, if any. Its own field rather than one of
	// the above: it decides where the others come from, so it cannot itself come from there.
	configFile *cliconf.Flags

	input    *string
	output   *string
	format   *string
	compact  *bool
	validate *bool

	color   *string
	quiet   *bool
	verbose *bool
	failOn  *string

	version *bool

	stdout         io.Writer
	stderr         io.Writer
	resolvedFormat string
}

func NewWithFlags(fs *flag.FlagSet, stdout, stderr io.Writer) *Config {
	return &Config{
		set:        fs,
		scan:       cliopts.Register(fs),
		configFile: cliconf.Register(fs),

		input: fs.String("input", "",
			"a specification to merge the scan's discoveries into, as JSON or YAML"),
		output: fs.String("output", "-", `write the specification here ("-" for standard output)`),
		format: fs.String("format", FormatAuto,
			`what to write: "json", "yaml", or "auto" to read the extension of -output`),
		compact: fs.Bool("compact", false, "write the document without indentation"),
		validate: fs.Bool("validate", false,
			"check the document against the Swagger 2.0 schema and report what is wrong with it"),

		color: fs.String("color", ColorAuto,
			`colored diagnostics: "always", "never", or "auto" for whenever stderr is a terminal`),
		quiet:   fs.Bool("quiet", false, "say nothing on standard error"),
		verbose: fs.Bool("verbose", false, "also report hints, which are muted by default"),
		failOn: fs.String("fail-on", FailNever,
			`exit non-zero when anything reported - by the scan or by -validate - reaches this`+"\n"+
				`severity: "error", "warning", or "never"`),

		version: fs.Bool("version", false, "print the version and exit"),
		stdout:  stdout,
		stderr:  stderr,
	}
}

func (c *Config) WantsVersion() bool {
	return c.version != nil && *c.version
}

func (c *Config) WantsValidation() bool {
	return c.validate != nil && *c.validate
}

func (c *Config) WantsCompactJSON() bool {
	return c.compact != nil && *c.compact
}

func (c *Config) Format() string {
	return c.resolvedFormat
}

func (c *Config) Output() string {
	return conv.Value(c.output)
}

// PrepareScan converts a CLI [Config] into proper [scanner.Options] that the scanner understands.
//
// It also prepares a configured [diagnostics.Reporter] for further reporting about diagnostics.
// At this point, no downstream consumer needs to know where a value came from (file, env, flag...).
func (c *Config) PrepareScan() (options *scanner.Options, reporter *diagnostics.Reporter, err error) {
	// 1. Load the configuration file if available: this presets the flags before parsing the command line.
	applied, configPath, err := configured(c.set, c.configFile)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", sentinel.ErrUsage, err)
	}

	// 2. Handle flags that don't pertain to the scanner: logging and output file
	c.resolvedFormat, err = resolveFormat(c.format, c.output)
	if err != nil {
		return nil, nil, err
	}

	colorized, err := resolveColor(c.color, c.stderr)
	if err != nil {
		return nil, nil, err
	}

	failOnSeverity, failing, err := resolveFailOn(c.failOn)
	if err != nil {
		return nil, nil, err
	}

	// 3. Configure a reporter that sets logging and failure modes
	reporter = diagnostics.NewReporter(diagnostics.ReporterConfig{
		Quiet:         conv.Value(c.quiet),
		Verbose:       conv.Value(c.verbose),
		Color:         colorized,
		FailThreshold: failOnSeverity,
		Failing:       failing,
		Stderr:        c.stderr,
	})
	reporter.AboutConfiguration(configPath, applied)

	// 4. Converts all options: values that pertain to the scanner are checked by cliopts.
	options, err = c.options(c.set.Args(), reporter)
	if err != nil {
		return nil, nil, err
	}

	return options, reporter, nil
}

// options assembles what the scan runs with.
func (c *Config) options(patterns []string, report *diagnostics.Reporter) (*codescan.Options, error) {
	opts := &codescan.Options{Packages: cliopts.Patterns(patterns)}
	if err := c.scan.Apply(opts); err != nil {
		return nil, errors.Join(err, sentinel.ErrUsage)
	}

	if path := c.input; path != nil && *path != "" {
		input, err := loadInputSpec(*path)
		if err != nil {
			return nil, err
		}
		opts.InputSpec = input
	}

	// Positions are reported relative to where the scan ran, which has to be settled before anything
	// is reported - and resolved, since -workdir is free to be relative but a diagnostic is not.
	root, err := filepath.Abs(opts.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve -workdir %q: %w", opts.WorkDir, err)
	}
	report.Root = root
	opts.OnDiagnostic = report.OnDiagnostic

	return opts, nil
}

// resolveColor decides whether to colorize.
//
// The mode is driven by config. The "auto" mode enables colorization only on a terminal.
//
// Environment variables NO_COLOR or TERM="dumb" disable colorization.
func resolveColor(modeOrNil *string, stderr io.Writer) (bool, error) {
	mode := conv.Value(modeOrNil)

	switch mode {
	case ColorAlways:
		return true, nil
	case ColorNever:
		return false, nil
	case ColorAuto:
		return isTerminal(stderr) && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb", nil
	default:
		return false, fmt.Errorf("%w: -color %q is not one of %s, %s, %s",
			sentinel.ErrUsage, mode, ColorAlways, ColorNever, ColorAuto)
	}
}

// resolveFailOn reads the threshold as the severity it names.
//
// Reported as a [codescan.Severity] and a flag saying whether there is a threshold at all,
// rather than as a severity meaning "none".
func resolveFailOn(failOnOrNil *string) (codescan.Severity, bool, error) {
	// NOTE(maintainers): the [codescan.Severity] enum belongs to the scanner.
	// The CLI interprets it as a threshold, so it adds a "never" level by adding a bool flag.
	// It would be better to add a grammar.SeverityNone for the zero value.
	failOn := conv.Value(failOnOrNil)

	switch failOn {
	case FailNever:
		return 0, false, nil
	case FailError:
		return codescan.SeverityError, true, nil
	case FailWarning:
		return codescan.SeverityWarning, true, nil
	default:
		return 0, false, fmt.Errorf("%w: -fail-on %q is not one of %s, %s, %s",
			sentinel.ErrUsage, failOn, FailError, FailWarning, FailNever)
	}
}

// resolveFormat decides what to write, from what was asked and where it is going.
//
// The extension is consulted only where nothing was said: a caller who writes -format=json to a file is being
// explicitly asking for a JSON output. If not specified, the output is inferred from the extension of the input.
func resolveFormat(formatOrNil, outputOrNil *string) (string, error) {
	format := conv.Value(formatOrNil)

	switch format {
	case FormatJSON, FormatYAML:
		return format, nil
	case FormatAuto:
		output := conv.Value(outputOrNil)
		if isYAMLName(output) {
			return FormatYAML, nil
		}

		return FormatJSON, nil
	default:
		return "", fmt.Errorf("%w: -format %q is not one of %s, %s, %s",
			sentinel.ErrUsage, format, FormatJSON, FormatYAML, FormatAuto)
	}
}

// isYAMLName reports whether a path names a YAML file.
func isYAMLName(path string) bool {
	lower := strings.ToLower(path)

	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}

// isTerminal detects if diagnostics are directed to a terminal.
func isTerminal(stderr io.Writer) bool {
	file, isFile := stderr.(*os.File)
	if !isFile || file == nil {
		return false
	}

	return term.IsTerminal(file.Fd())
}
