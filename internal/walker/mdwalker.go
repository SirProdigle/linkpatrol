package walker

import (
	"bufio"
	"context"
	"os"
	"strings"

	"github.com/sirprodigle/linkpatrol/internal/cache"
)

type MarkdownWalker struct {
	cache   *cache.Cache
	results chan<- WalkerResult
}

func NewMarkdownWalker(cache *cache.Cache, results chan<- WalkerResult) *MarkdownWalker {
	return &MarkdownWalker{
		cache:   cache,
		results: results,
	}
}

func (w *MarkdownWalker) Walk(ctx context.Context, uri string) error {
	f, err := os.Open(uri)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()

		// Extract internal MD file references first (for further walking)
		for _, match := range InternalMdRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: uri,
					Path:     match[1],
					Type:     PathTypeRelativeFile,
				}
			}
		}

		// Extract HTTP URLs
		for _, match := range HttpUrlRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: uri,
					Path:     match[1],
					Type:     PathTypeUrl,
				}
			}
		}

		// Extract markdown links (excluding internal MD files already captured)
		for _, match := range MarkdownLinkRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				// Skip if it's an internal MD file (already captured separately)
				if !InternalMdRegex.MatchString("[](" + match[1] + ")") {
					w.results <- WalkerResult{
						BasePath: uri,
						Path:     match[1],
						Type:     PathTypeUrl,
					}
				}
			}
		}

		// Extract image links
		for _, match := range ImageLinkRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				if strings.HasPrefix(match[1], "http://") || strings.HasPrefix(match[1], "https://") {
					w.results <- WalkerResult{
						BasePath: uri,
						Path:     match[1],
						Type:     PathTypeUrl,
					}
				} else {
					w.results <- WalkerResult{
						BasePath: uri,
						Path:     match[1],
						Type:     PathTypeRelativeFile,
					}
				}
			}
		}

		// Extract reference links
		for _, match := range ReferenceLinkRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: uri,
					Path:     match[1],
					Type:     PathTypeUrl,
				}
			}
		}

		// Extract reference definitions
		for _, match := range ReferenceDefRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: uri,
					Path:     match[1],
					Type:     PathTypeUrl,
				}
			}
		}

		// Extract bare URLs
		for _, match := range BareUrlRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: uri,
					Path:     match[1],
					Type:     PathTypeUrl,
				}
			}
		}

		// Extract email links
		for _, match := range EmailRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: uri,
					Path:     match[1],
					Type:     PathTypeEmail,
				}
			}
		}

		// Extract FTP URLs
		for _, match := range FtpRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: uri,
					Path:     match[1],
					Type:     PathTypeFtp,
				}
			}
		}

		// Extract Git URLs
		for _, match := range GitRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: uri,
					Path:     match[1],
					Type:     PathTypeGit,
				}
			}
		}

		// Extract file URLs
		for _, match := range FileRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: uri,
					Path:     match[1],
					Type:     PathTypeFile,
				}
			}
		}

		// Extract relative paths
		for _, match := range RelativePathRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				if strings.HasPrefix(match[1], "http://") || strings.HasPrefix(match[1], "https://") {
					w.results <- WalkerResult{
						BasePath: uri,
						Path:     match[1],
						Type:     PathTypeUrl,
					}
				} else {
					w.results <- WalkerResult{
						BasePath: uri,
						Path:     match[1],
						Type:     PathTypeRelativeFile,
					}
				}
			}
		}
	}

	return nil
}
