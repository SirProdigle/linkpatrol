package walker

import (
	"bufio"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirprodigle/linkpatrol/internal/cache"
)

type HtmlWalker struct {
	cache   *cache.Cache
	client  *http.Client
	results chan<- WalkerResult
}

func NewHtmlWalker(cache *cache.Cache, timeout time.Duration, results chan<- WalkerResult) *HtmlWalker {
	return &HtmlWalker{
		cache: cache,
		client: &http.Client{
			Timeout: timeout,
		},
		results: results,
	}
}

func (w *HtmlWalker) Walk(ctx context.Context, uri string) error {
	f, err := os.Open(uri)
	if err != nil {
		return err
	}
	defer f.Close()

	// Extract the base directory from the URI for resolving relative paths
	basePath := filepath.Dir(uri)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Check for HTML links first
		for _, match := range HtmlLinkRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				href := match[1]
				// Categorize based on the href content
				switch {
				case strings.HasPrefix(href, "mailto:"):
					w.results <- WalkerResult{
						BasePath: basePath,
						Path:     href,
						Type:     PathTypeEmail,
					}
				case strings.HasPrefix(href, "tel:"):
					w.results <- WalkerResult{
						BasePath: basePath,
						Path:     href,
						Type:     PathTypeTel,
					}
				case strings.HasPrefix(href, "#"):
					w.results <- WalkerResult{
						BasePath: basePath,
						Path:     href,
						Type:     PathTypeAnchor,
					}
				case href == "/":
					w.results <- WalkerResult{
						BasePath: basePath,
						Path:     href,
						Type:     PathTypeRoot,
					}
				case strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://"):
					w.results <- WalkerResult{
						BasePath: basePath,
						Path:     href,
						Type:     PathTypeUrl,
					}
				default:
					// Relative path - could be file or URL depending on context
					w.results <- WalkerResult{
						BasePath: basePath,
						Path:     href,
						Type:     PathTypeRelativeFile,
					}
				}
			}
		}

		// Check for HTML images
		for _, match := range HtmlImgRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				src := match[1]
				if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
					w.results <- WalkerResult{
						BasePath: basePath,
						Path:     src,
						Type:     PathTypeUrl,
					}
				} else {
					w.results <- WalkerResult{
						BasePath: basePath,
						Path:     src,
						Type:     PathTypeRelativeFile,
					}
				}
			}
		}

		// Check for bare HTTP URLs
		for _, match := range HttpUrlRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: basePath,
					Path:     match[1],
					Type:     PathTypeUrl,
				}
			}
		}

		// Check for bare URLs (should be same as HTTP URLs, but keeping for compatibility)
		for _, match := range BareUrlRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: basePath,
					Path:     match[1],
					Type:     PathTypeUrl,
				}
			}
		}

		// Check for email links
		for _, match := range EmailRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: basePath,
					Path:     match[1],
					Type:     PathTypeEmail,
				}
			}
		}

		// Check for telephone links
		for _, match := range TelRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: basePath,
					Path:     match[1],
					Type:     PathTypeTel,
				}
			}
		}

		// Check for anchor links
		for _, match := range AnchorRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: basePath,
					Path:     match[1],
					Type:     PathTypeAnchor,
				}
			}
		}

		// Check for root path
		for _, match := range RootPathRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: basePath,
					Path:     match[1],
					Type:     PathTypeRoot,
				}
			}
		}

		// Check for FTP links
		for _, match := range FtpRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: basePath,
					Path:     match[1],
					Type:     PathTypeFtp,
				}
			}
		}

		// Check for Git links
		for _, match := range GitRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: basePath,
					Path:     match[1],
					Type:     PathTypeGit,
				}
			}
		}

		// Check for file links
		for _, match := range FileRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: basePath,
					Path:     match[1],
					Type:     PathTypeFile,
				}
			}
		}

		// Check for relative paths (only if they look like files)
		for _, match := range RelativePathRegex.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				w.results <- WalkerResult{
					BasePath: basePath,
					Path:     match[1],
					Type:     PathTypeRelativeFile,
				}
			}
		}

	}
	return nil
}
