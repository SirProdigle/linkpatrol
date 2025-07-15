package walker

import "regexp"

type RegexIdentifier int

const (
	HtmlATagRegexIdentifier RegexIdentifier = iota
	HtmlImageRegexIdentifier
	HtmlScriptRegexIdentifier
	HtmlStyleRegexIdentifier
	HtmlLinkRegexIdentifier
	ImgSrcsetRegexIdentifier
	CssImportRegexIdentifier
	CssUrlRegexIdentifier
	JsonUrlRegexIdentifier
	ImgDataSrcRegexIdentifier
	ImgLazySrcRegexIdentifier
	EmailRegexIdentifier
	TelRegexIdentifier
	HttpUrlRegexIdentifier
	RelativeUrlRegexIdentifier
)

var RegexIdentifiers = map[RegexIdentifier]*regexp.Regexp{
	HtmlATagRegexIdentifier:    HtmlATagRegex,
	HtmlImageRegexIdentifier:   HtmlImageRegex,
	HtmlScriptRegexIdentifier:  HtmlScriptRegex,
	HtmlStyleRegexIdentifier:   HtmlStyleRegex,
	HtmlLinkRegexIdentifier:    HtmlLinkRegex,
	ImgSrcsetRegexIdentifier:   ImgSrcsetRegex,
	CssImportRegexIdentifier:   CssImportRegex,
	CssUrlRegexIdentifier:      CssUrlRegex,
	JsonUrlRegexIdentifier:     JsonUrlRegex,
	ImgDataSrcRegexIdentifier:  ImgDataSrcRegex,
	ImgLazySrcRegexIdentifier:  ImgLazySrcRegex,
	EmailRegexIdentifier:       EmailRegex,
	TelRegexIdentifier:         TelRegex,
	HttpUrlRegexIdentifier:     HttpUrlRegex,
	RelativeUrlRegexIdentifier: RelativeUrlRegex,
}

const (

	// HTML link patterns for web scraping - non-greedy for multiline support
	htmlATagPattern   = `<a[^>]*?href=["'""]([^"'""]+?)["'""][^>]*?>`
	htmlImagePattern  = `<img[^>]*?src=["'""]([^"'""]+?)["'""][^>]*?>`
	htmlScriptPattern = `<script[^>]*?src=["'""]([^"'""]+?)["'""][^>]*?>`
	htmlStylePattern  = `<style[^>]*?href=["'""]([^"'""]+?)["'""][^>]*?>`
	htmlLinkPattern   = `<link[^>]*?href=["'""]([^"'""]+?)["'""][^>]*?>`
	
	// Additional patterns for comprehensive coverage - multiline aware
	imgSrcsetPattern     = `srcset=["'""]([^"'""]+?)["'""]`
	cssImportPattern     = `@import\s+["'""]([^"'""]+?)["'""]`
	
	// CSS url() patterns for background images, fonts, etc.
	cssUrlPattern        = `url\(["']?(https?://[^\s"')\],;]+)["']?\)`
	
	// JSON-LD and script content URL patterns
	jsonUrlPattern       = `["'""]([^"'""]*https?://[^\s"'""]+)["'""]`
	
	// Alternative image patterns for edge cases  
	imgDataSrcPattern    = `data-src=["'""]([^"'""]+?)["'""]`
	imgLazySrcPattern    = `<img[^>]*?data-lazy-src=["'""]([^"'""]+?)["'""][^>]*?>`

	// RAW HTTP/HTTPS URLs - stop at HTML tags, quotes, whitespace, and brackets
	httpUrlPattern     = `(https?://[^\s\)\]"'<>{}|\\^` + "`" + `]+)`
	relativeUrlPattern = `href=["']([^"']+)["']`

	// Email and other protocol links
	emailPattern = `(mailto:[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`
	telPattern   = `(tel:[+]?[0-9\-\(\)\s]+)`
)

var (
	// Compiled regex patterns for web scraping
	HtmlATagRegex     = regexp.MustCompile(htmlATagPattern)
	HtmlImageRegex    = regexp.MustCompile(htmlImagePattern)
	HtmlScriptRegex   = regexp.MustCompile(htmlScriptPattern)
	HtmlStyleRegex    = regexp.MustCompile(htmlStylePattern)
	HtmlLinkRegex     = regexp.MustCompile(htmlLinkPattern)
	ImgSrcsetRegex    = regexp.MustCompile(imgSrcsetPattern)
	CssImportRegex    = regexp.MustCompile(cssImportPattern)
	CssUrlRegex       = regexp.MustCompile(cssUrlPattern)
	JsonUrlRegex      = regexp.MustCompile(jsonUrlPattern)
	ImgDataSrcRegex   = regexp.MustCompile(imgDataSrcPattern)
	ImgLazySrcRegex   = regexp.MustCompile(imgLazySrcPattern)
	EmailRegex        = regexp.MustCompile(emailPattern)
	TelRegex          = regexp.MustCompile(telPattern)
	HttpUrlRegex      = regexp.MustCompile(httpUrlPattern)
	RelativeUrlRegex  = regexp.MustCompile(relativeUrlPattern)
)

func GetRegexes() map[RegexIdentifier]*regexp.Regexp {
	return RegexIdentifiers
}
