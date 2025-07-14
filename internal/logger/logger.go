package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

// Level represents the logging level
type Level int

const (
	LevelError Level = iota
	LevelWarn
	LevelInfo
	LevelDebug
)

// ANSI color codes
const (
	colorReset   = "\033[0m"
	colorBold    = "\033[1m"
	colorDim     = "\033[2m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorGray    = "\033[90m"
)

// winsize represents terminal window size
type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

// Logger provides structured logging with verbosity control
type Logger struct {
	out        io.Writer
	errOut     io.Writer
	verbose    bool
	termWidth  int
	noTruncate bool
}

// Option configures a Logger
type Option func(*Logger)

// WithOutput sets the output writer
func WithOutput(w io.Writer) Option {
	return func(l *Logger) {
		l.out = w
	}
}

// WithErrorOutput sets the error output writer
func WithErrorOutput(w io.Writer) Option {
	return func(l *Logger) {
		l.errOut = w
	}
}

// WithTerminalWidth sets a custom terminal width
func WithTerminalWidth(width int) Option {
	return func(l *Logger) {
		l.termWidth = width
	}
}

// WithNoTruncate disables text truncation
func WithNoTruncate(noTruncate bool) Option {
	return func(l *Logger) {
		l.noTruncate = noTruncate
	}
}

// getTerminalWidth detects the current terminal width
func getTerminalWidth() int {
	ws := &winsize{}
	retCode, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 {
		// Fall back to a reasonable default if detection fails
		return 120
	}

	if errno != 0 || ws.Col == 0 {
		return 120
	}

	return int(ws.Col)
}

// New creates a new logger instance
func New(verbose bool, opts ...Option) *Logger {
	l := &Logger{
		out:       os.Stdout,
		errOut:    os.Stderr,
		verbose:   verbose,
		termWidth: getTerminalWidth(),
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// IsVerbose returns whether verbose logging is enabled
func (l *Logger) IsVerbose() bool {
	return l.verbose
}

// Info logs an informational message (always shown)
func (l *Logger) Info(msg string, args ...any) {
	l.log(l.out, "ℹ️", colorCyan, msg, args...)
}

// Success logs a success message (always shown)
func (l *Logger) Success(msg string, args ...any) {
	l.log(l.out, "✅", colorGreen, msg, args...)
}

// Warn logs a warning message (always shown)
func (l *Logger) Warn(msg string, args ...any) {
	l.log(l.out, "⚠️", colorYellow, msg, args...)
}

// Error logs an error message (always shown)
func (l *Logger) Error(msg string, args ...any) {
	l.log(l.errOut, "❌", colorRed, msg, args...)
}

// Debug logs a debug message (only in verbose mode)
func (l *Logger) Debug(msg string, args ...any) {
	if l.verbose {
		l.log(l.out, "🐛", colorGray, msg, args...)
	}
}

// Progress logs a progress message (only in verbose mode)
func (l *Logger) Progress(msg string, args ...any) {
	if l.verbose {
		l.log(l.out, "⏳", colorCyan, msg, args...)
	}
}

// Trace logs a trace message (only in verbose mode)
func (l *Logger) Trace(msg string, args ...any) {
	if l.verbose {
		l.log(l.out, "🔍", colorDim, msg, args...)
	}
}

// StartSection prints a section header
func (l *Logger) StartSection(title string) {
	// Calculate padding to fill terminal width
	headerText := "🚀 " + title + " "
	remainingWidth := max(0, l.termWidth-len(headerText)-2) // -2 for newlines/margins
	padding := strings.Repeat("=", remainingWidth)
	fmt.Fprintf(l.out, "\n%s%s%s%s\n", colorBold+colorBlue, headerText, padding, colorReset)
}

// FileWalk logs file discovery (only in verbose mode)
func (l *Logger) FileWalk(fileType, path string) {
	if l.verbose {
		l.log(l.out, "📄", colorCyan, "Found %s: %s", fileType, path)
	}
}

// LinkTest logs link testing progress (only in verbose mode)
func (l *Logger) LinkTest(url, status string) {
	if !l.verbose {
		return
	}

	var emoji, color string
	switch status {
	case "testing":
		emoji, color = "🧪", colorCyan
	case "success":
		emoji, color = "✅", colorGreen
	case "failed":
		emoji, color = "❌", colorRed
	case "timeout":
		emoji, color = "⏰", colorYellow
	default:
		emoji, color = "🔗", colorCyan
	}

	l.log(l.out, emoji, color, "%s: %s", status, url)
}

// RateLimit logs rate limiting info (only in verbose mode)
func (l *Logger) RateLimit(domain string, rate int, action string) {
	if l.verbose {
		l.log(l.out, "🚦", colorMagenta, "%s for %s (rate: %d req/s)", action, domain, rate)
	}
}

// Config logs configuration information
func (l *Logger) Config(dir string, watch bool, concurrency int, timeout, rateLimit any) {
	if !l.verbose {
		return
	}

	l.log(l.out, "🔧", colorBlue, "Configuration")
	fmt.Fprintf(l.out, "  %sDirectory:%s %s\n", colorCyan, colorReset, dir)
	fmt.Fprintf(l.out, "  %sWatch:%s %t\n", colorCyan, colorReset, watch)
	fmt.Fprintf(l.out, "  %sWalker concurrency:%s %d\n", colorCyan, colorReset, concurrency)
	fmt.Fprintf(l.out, "  %sTester concurrency:%s %d\n", colorCyan, colorReset, concurrency)
	fmt.Fprintf(l.out, "  %sTimeout:%s %v\n", colorCyan, colorReset, timeout)
	fmt.Fprintf(l.out, "  %sRate limit:%s %v req/s\n", colorCyan, colorReset, rateLimit)
}

// Watch logs watch mode status
func (l *Logger) Watch(dir string) {
	l.log(l.out, "👀", colorBlue, "Watching %s for changes...", dir)
}

// FileChange logs file change events (only in verbose mode)
func (l *Logger) FileChange(path string) {
	if l.verbose {
		l.log(l.out, "📝", colorYellow, "File changed: %s", path)
	}
}

// WatchError logs watcher errors
func (l *Logger) WatchError(err error) {
	l.log(l.errOut, "❌", colorRed, "Watcher error: %v", err)
}

// FilesFound logs the discovery of markdown files
func (l *Logger) FilesFound(mdFiles, htmlFiles int) {
	l.log(l.out, "📊", colorBlue, "Found %d markdown files", mdFiles)
}

// TestResults logs the final link testing results
func (l *Logger) TestResults(deadLinks, timeoutLinks int) {
	if deadLinks > 0 || timeoutLinks > 0 {
		l.log(l.errOut, "💀", colorRed, "Found %d dead links and %d timeout links", deadLinks, timeoutLinks)
	} else {
		l.log(l.out, "✨", colorGreen, "All links are working!")
	}
}

// Shutdown logs shutdown message
func (l *Logger) Shutdown() {
	l.log(l.out, "🛑", colorYellow, "Shutdown signal received, stopping...")
}

// Waiting logs waiting message (only in verbose mode)
func (l *Logger) Waiting() {
	if l.verbose {
		l.log(l.out, "⏸️", colorCyan, "Waiting for file changes...")
	}
}

// DisplayEntry represents a cache entry formatted for display
type DisplayEntry struct {
	URL    string
	Status string
	Emoji  string
	Error  string
	Color  string
}

// CacheTable displays cache entries in a formatted table
func (l *Logger) CacheTable(entries []DisplayEntry) {
	if len(entries) == 0 {
		l.log(l.out, "📭", colorBlue, "No entries in cache")
		return
	}

	// Calculate dynamic column widths based on terminal width
	const statusColWidth = 8 // "TIMEOUT" = 7 chars + padding
	const emojiColWidth = 6  // Emoji + padding
	const minUrlWidth = 30   // Minimum URL width
	const minErrorWidth = 15 // Minimum error width
	const padding = 6        // Space for separators and padding

	// Calculate available space for URL and Error columns (70:30 split)
	fixedWidth := statusColWidth + emojiColWidth + padding
	availableWidth := l.termWidth - fixedWidth

	// Split remaining space 70:30 between URL and Error
	urlColWidth := max(minUrlWidth, (availableWidth*50)/100)
	errorColWidth := max(minErrorWidth, availableWidth-urlColWidth)

	// Header
	fmt.Fprintf(l.out, "%s%-*s %-*s %-*s %-*s%s\n",
		colorCyan, urlColWidth, "URL", statusColWidth, "Status", emojiColWidth, "Emoji", errorColWidth, "Error", colorReset)
	fmt.Fprintf(l.out, "%s%s%s\n", colorGray, strings.Repeat("─", l.termWidth), colorReset)

	// Entries
	for _, entry := range entries {
		errorMsg := entry.Error
		if errorMsg == "" {
			errorMsg = "-"
		}

		url := entry.URL
		if l.noTruncate {
			// No truncation - use actual content lengths
			fmt.Fprintf(l.out, "%s%s %s %s %s%s\n",
				entry.Color, url, entry.Status, entry.Emoji, errorMsg, colorReset)
		} else {
			// Truncate URL if too long
			if len(url) > urlColWidth-1 {
				url = url[:urlColWidth-4] + "..."
			}

			// Truncate error message if too long
			if len(errorMsg) > errorColWidth-1 {
				errorMsg = errorMsg[:errorColWidth-4] + "..."
			}

			fmt.Fprintf(l.out, "%s%-*s %-*s %-*s %-*s%s\n",
				entry.Color, urlColWidth, url, statusColWidth, entry.Status, emojiColWidth, entry.Emoji, errorColWidth, errorMsg, colorReset)
		}
	}

	// Footer
	fmt.Fprintf(l.out, "%s%s%s\n", colorGray, strings.Repeat("─", l.termWidth), colorReset)
	l.log(l.out, "📊", colorBold, "Total entries: %d", len(entries))
}

// log is the internal logging method that handles formatting
func (l *Logger) log(w io.Writer, emoji, color, msg string, args ...any) {
	formattedMsg := fmt.Sprintf(msg, args...)
	fmt.Fprintf(w, "%s%s %s%s\n", color, emoji, formattedMsg, colorReset)
}
