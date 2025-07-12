package walker

import "regexp"

const (
	// External HTTP/HTTPS URLs (for validation)
	httpUrlPattern = `(https?://[^\s\)\]"']+)`

	// Markdown link patterns that extract the URL
	markdownLinkPattern  = `\[[^\]]+\]\(([^)]+)\)`   // [text](url) - captures URL in group 1
	imageLinkPattern     = `!\[[^\]]*\]\(([^)]+)\)`  // ![alt](url) - captures URL in group 1
	referenceLinkPattern = `\[[^\]]+\]\[([^\]]*)\]`  // [text][ref] - captures ref in group 1
	referenceDefPattern  = `^\[[^\]]+\]:\s*([^\s]+)` // [ref]: url - captures URL in group 1

	// Bare URLs (not wrapped in markdown syntax)
	bareUrlPattern = `(https?://[^\s\)\]"']+)`

	// Email links (for validation)
	emailPattern = `(mailto:[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`

	// Telephone links
	telPattern = `(tel:[+]?[0-9\-\(\)\s]+)`

	// Anchor links (fragment identifiers) - exclude CSS color codes
	// Must start with # followed by a letter, then letters, numbers, hyphens, or underscores
	// Excludes hex color codes by requiring at least one letter beyond a-f anywhere
	anchorPattern = `(#[a-zA-Z][a-zA-Z0-9\-_]*[g-zG-Z][a-zA-Z0-9\-_]*)`

	// Root path
	rootPathPattern = `^/$`

	// Other protocols that should be validated
	ftpPattern  = `(ftp://[^\s\)\]"']+)`
	gitPattern  = `((?:git|ssh)://[^\s\)\]"']+)`
	filePattern = `(file://[^\s\)\]"']+)`

	// Internal markdown file references
	internalMdPattern = `\[[^\]]+\]\(([^)]*\.(?:md|markdown))\)` // [text](file.md) - captures .md/.markdown files

	// Relative paths that might be external (depending on context)
	relativePathPattern = `(\./[a-zA-Z0-9\-_/]+\.(md|markdown|txt|jpg|jpeg|png|gif|svg|webp|pdf|doc|docx|xls|xlsx|ppt|pptx|mp4|mp3|wav|ogg|webm|mov|avi|wmv|flv|mkv|m4v|m4a|aac)$)`
)

var (
	// Compiled regex patterns
	HttpUrlRegex       = regexp.MustCompile(httpUrlPattern)
	MarkdownLinkRegex  = regexp.MustCompile(markdownLinkPattern)
	ImageLinkRegex     = regexp.MustCompile(imageLinkPattern)
	ReferenceLinkRegex = regexp.MustCompile(referenceLinkPattern)
	ReferenceDefRegex  = regexp.MustCompile(referenceDefPattern)
	BareUrlRegex       = regexp.MustCompile(bareUrlPattern)
	EmailRegex         = regexp.MustCompile(emailPattern)
	TelRegex           = regexp.MustCompile(telPattern)
	AnchorRegex        = regexp.MustCompile(anchorPattern)
	RootPathRegex      = regexp.MustCompile(rootPathPattern)
	InternalMdRegex    = regexp.MustCompile(internalMdPattern)
	RelativePathRegex  = regexp.MustCompile(relativePathPattern)
	FtpRegex           = regexp.MustCompile(ftpPattern)
	GitRegex           = regexp.MustCompile(gitPattern)
	FileRegex          = regexp.MustCompile(filePattern)
)
