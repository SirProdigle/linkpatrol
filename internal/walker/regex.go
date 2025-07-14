package walker

import "regexp"

type RegexIdentifier int

const (
	HtmlATagRegexIdentifier RegexIdentifier = iota
	HtmlImageRegexIdentifier
	HtmlScriptRegexIdentifier
	HtmlStyleRegexIdentifier
	HtmlMetaRegexIdentifier
	HtmlLinkRegexIdentifier
	EmailRegexIdentifier
	TelRegexIdentifier
	HttpUrlRegexIdentifier
)

var RegexIdentifiers = map[RegexIdentifier]*regexp.Regexp{
	HtmlATagRegexIdentifier:   HtmlATagRegex,
	HtmlImageRegexIdentifier:  HtmlImageRegex,
	HtmlScriptRegexIdentifier: HtmlScriptRegex,
	HtmlStyleRegexIdentifier:  HtmlStyleRegex,
	HtmlMetaRegexIdentifier:   HtmlMetaRegex,
	HtmlLinkRegexIdentifier:   HtmlLinkRegex,
	EmailRegexIdentifier:      EmailRegex,
	TelRegexIdentifier:        TelRegex,
	HttpUrlRegexIdentifier:    HttpUrlRegex,
}

const (

	// HTML link patterns for web scraping
	htmlATagPattern   = `<a[^>]+href=["'""]([^"'""]+)["'""][^>]*>`
	htmlImagePattern  = `<img[^>]+src=["'""]([^"'""]+)["'""][^>]*>`
	htmlScriptPattern = `<script[^>]+src=["'""]([^"'""]+)["'""][^>]*>`
	htmlStylePattern  = `<style[^>]+href=["'""]([^"'""]+)["'""][^>]*>`
	htmlMetaPattern   = `<meta[^>]+content=["'""]([^"'""]+)["'""][^>]*>`
	htmlLinkPattern   = `<link[^>]+href=["'""]([^"'""]+)["'""][^>]*>`

	// RAW HTTP/HTTPS URLs
	httpUrlPattern = `(https?://[^\s\)\]"']+)`

	// Email and other protocol links
	emailPattern = `(mailto:[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`
	telPattern   = `(tel:[+]?[0-9\-\(\)\s]+)`
)

var (
	// Compiled regex patterns for web scraping
	HtmlATagRegex   = regexp.MustCompile(htmlATagPattern)
	HtmlImageRegex  = regexp.MustCompile(htmlImagePattern)
	HtmlScriptRegex = regexp.MustCompile(htmlScriptPattern)
	HtmlStyleRegex  = regexp.MustCompile(htmlStylePattern)
	HtmlMetaRegex   = regexp.MustCompile(htmlMetaPattern)
	HtmlLinkRegex   = regexp.MustCompile(htmlLinkPattern)
	EmailRegex      = regexp.MustCompile(emailPattern)
	TelRegex        = regexp.MustCompile(telPattern)
	HttpUrlRegex    = regexp.MustCompile(httpUrlPattern)
)

func GetRegexes() map[RegexIdentifier]*regexp.Regexp {
	return RegexIdentifiers
}
