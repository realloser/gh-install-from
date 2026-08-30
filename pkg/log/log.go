package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Logger is the global logger instance
	Logger *slog.Logger

	// verbose controls whether debug logs are enabled
	verbose bool
)

func init() {
	// Initialize with default settings
	Init(false)
}

// Init initializes the logger with the given verbosity level
func Init(isVerbose bool) {
	verbose = isVerbose

	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := newCLIHandler(os.Stderr, level)
	Logger = slog.New(handler)

	// Set as default logger so raw slog.* calls also use the custom handler.
	slog.SetDefault(Logger)
}

// IsVerbose returns whether verbose logging is enabled
func IsVerbose() bool {
	return verbose
}

// Debug logs at debug level if verbose mode is enabled
func Debug(msg string, args ...any) {
	if verbose {
		Logger.Debug(msg, args...)
	}
}

// Info logs at info level
func Info(msg string, args ...any) {
	Logger.Info(msg, args...)
}

// Warn logs at warn level
func Warn(msg string, args ...any) {
	Logger.Warn(msg, args...)
}

// Error logs at error level
func Error(msg string, args ...any) {
	Logger.Error(msg, args...)
}

// --- custom handler ---

// cliHandler renders INFO/WARN/ERROR as clean human-readable lines (a styled
// prefix + message + inline key=value args) and DEBUG as structured slog text.
// All output goes to stderr so stdout stays clean for machine-readable output.
type cliHandler struct {
	w       io.Writer
	level   slog.Leveler
	verbose bool
	attrs   []slog.Attr
	groups  []string
}

func newCLIHandler(w io.Writer, level slog.Leveler) *cliHandler {
	return &cliHandler{w: w, level: level, verbose: level.Level() <= slog.LevelDebug}
}

func (h *cliHandler) Enabled(_ context.Context, lvl slog.Level) bool {
	return lvl >= h.level.Level()
}

func (h *cliHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nh := *h
	nh.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &nh
}

func (h *cliHandler) WithGroup(name string) slog.Handler {
	nh := *h
	nh.groups = append(append([]string(nil), h.groups...), name)
	return &nh
}

func (h *cliHandler) Handle(_ context.Context, r slog.Record) error {
	// DEBUG keeps the structured text format (verbose diagnostics).
	if r.Level < slog.LevelInfo {
		return h.renderStructured(r)
	}
	return h.renderClean(r)
}

// renderClean prints a styled prefix + message + inline args, e.g.:
//
//	→ fetching latest release (repo=messerm/jenkins-job)
var (
	infoPrefix = lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Render("→")
	warnPrefix = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("⚠")
	errPrefix  = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render("✗")
)

func (h *cliHandler) renderClean(r slog.Record) error {
	var b strings.Builder

	switch {
	case r.Level >= slog.LevelError:
		b.WriteString(errPrefix)
	case r.Level >= slog.LevelWarn:
		b.WriteString(warnPrefix)
	default:
		b.WriteString(infoPrefix)
	}
	b.WriteByte(' ')
	b.WriteString(r.Message)

	// Inline key=value args (handler-level + record-level).
	args := append([]slog.Attr{}, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		args = append(args, a)
		return true
	})
	if len(args) > 0 {
		b.WriteString(" (")
		for i, a := range args {
			if i > 0 {
				b.WriteString(" ")
			}
			b.WriteString(a.Key)
			b.WriteByte('=')
			b.WriteString(attrValue(a.Value))
		}
		b.WriteByte(')')
	}
	b.WriteByte('\n')

	_, err := io.WriteString(h.w, b.String())
	return err
}

// renderStructured prints the classic slog text format for DEBUG lines.
func (h *cliHandler) renderStructured(r slog.Record) error {
	var b strings.Builder
	b.WriteString("level=")
	b.WriteString(r.Level.String())
	b.WriteString(" msg=")
	b.WriteString(strconv.Quote(r.Message))

	r.Attrs(func(a slog.Attr) bool {
		b.WriteByte(' ')
		b.WriteString(a.Key)
		b.WriteByte('=')
		b.WriteString(attrValue(a.Value))
		return true
	})
	b.WriteByte('\n')

	_, err := io.WriteString(h.w, b.String())
	return err
}

// attrValue renders a slog.Value as a compact string (quoting when needed).
func attrValue(v slog.Value) string {
	v = v.Resolve()
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindInt64:
		return strconv.FormatInt(v.Int64(), 10)
	case slog.KindUint64:
		return strconv.FormatUint(v.Uint64(), 10)
	case slog.KindFloat64:
		return strconv.FormatFloat(v.Float64(), 'f', -1, 64)
	case slog.KindBool:
		return strconv.FormatBool(v.Bool())
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindTime:
		return v.Time().String()
	default:
		return strconv.Quote(v.String())
	}
}
