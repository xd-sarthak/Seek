package crawler

import (
	"net/url"
	"regexp"
	"spider/internal/utils"
	"strings"

	"golang.org/x/net/html"
)

// PageDirectives holds robots-related directives extracted from
// <meta name="robots"> tags in the HTML document.
type PageDirectives struct {
	NoIndex  bool // Page should not be indexed/stored
	NoFollow bool // Links on this page should not be followed
}

// getURLsFromHTML parses an HTML document and extracts all hyperlinks and images,
// while also parsing <meta name="robots"> directives.
//
// Returns:
//   - links: deduplicated slice of absolute URLs extracted from <a href> tags
//     (links with rel="nofollow" are excluded)
//   - imagesMap: map of normalized image source URLs to their attributes
//     (keys: image URL, values: map with "src" and optionally "alt")
//   - directives: parsed robots meta directives (noindex, nofollow)
//   - err: non-nil if the base URL or HTML could not be parsed
//
// Relative URLs are resolved against rawURL. Malformed and non-ASCII URLs
// are silently skipped.
func getURLsFromHTML(htmlBody string, rawURL string) ([]string, map[string]map[string]string, PageDirectives, error) {
	baseURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, nil, PageDirectives{}, err
	}

	node, err := html.Parse(strings.NewReader(htmlBody))
	if err != nil {
		return nil, nil, PageDirectives{}, err
	}

	linksSet := make(map[string]struct{})
	imagesMap := make(map[string]map[string]string)
	directives := PageDirectives{}

	traverse(node, baseURL, linksSet, imagesMap, &directives)

	// Convert set to slice
	links := make([]string, 0, len(linksSet))
	for link := range linksSet {
		links = append(links, link)
	}

	return links, imagesMap, directives, nil
}

// nonASCIIRegex matches any character outside the printable ASCII range.
// Used to filter out URLs containing non-ASCII characters.
var nonASCIIRegex = regexp.MustCompile(`[^\x20-\x7E]`)

// traverse performs a depth-first walk of the HTML node tree, collecting
// <a href> links into linksSet, <img src/alt> data into imagesMap,
// and <meta name="robots"> directives into directives.
// Links with rel="nofollow" are skipped. Relative URLs are resolved against baseURL.
func traverse(node *html.Node, baseURL *url.URL, linksSet map[string]struct{}, imagesMap map[string]map[string]string, directives *PageDirectives) {
	if node == nil {
		return
	}

	if node.Type == html.ElementNode && node.Data == "meta" {
		// Check for <meta name="robots" content="...">
		var nameAttr, contentAttr string
		for _, attr := range node.Attr {
			switch strings.ToLower(attr.Key) {
			case "name":
				nameAttr = strings.ToLower(attr.Val)
			case "content":
				contentAttr = strings.ToLower(attr.Val)
			}
		}

		if nameAttr == "robots" && contentAttr != "" {
			parts := strings.Split(contentAttr, ",")
			for _, part := range parts {
				directive := strings.TrimSpace(part)
				switch directive {
				case "noindex":
					directives.NoIndex = true
				case "nofollow":
					directives.NoFollow = true
				case "none":
					directives.NoIndex = true
					directives.NoFollow = true
				}
			}
		}
	} else if node.Type == html.ElementNode && node.Data == "a" {
		// Check for rel="nofollow" on this specific link
		hasNoFollow := false
		var hrefVal string

		for _, attr := range node.Attr {
			if attr.Key == "href" {
				hrefVal = attr.Val
			}
			if attr.Key == "rel" {
				rels := strings.Fields(strings.ToLower(attr.Val))
				for _, r := range rels {
					if r == "nofollow" {
						hasNoFollow = true
					}
				}
			}
		}

		// Only add the link if it doesn't have rel="nofollow" and has an href
		if !hasNoFollow && hrefVal != "" {
			rawHref := hrefVal

			// Skip malformed URLs
			if !strings.ContainsAny(rawHref, " <>\"") {
				// Skip non-ASCII urls
				if !nonASCIIRegex.MatchString(rawHref) {
					u, err := url.Parse(rawHref)
					if err == nil {
						var resolved string
						if u.IsAbs() {
							resolved = u.String()
						} else {
							resolved = baseURL.ResolveReference(u).String()
						}
						linksSet[resolved] = struct{}{}
					}
				}
			}
		}
	} else if node.Type == html.ElementNode && node.Data == "img" {
		imageDetails := make(map[string]string)
		for _, attr := range node.Attr {
			if attr.Key == "src" {
				rawSrc := attr.Val

				// Skip malformed URLS
				if strings.ContainsAny(rawSrc, " <>\"") {
					continue
				}

				// Skip non-ASCII urls
				if nonASCIIRegex.MatchString(rawSrc) {
					continue
				}

				// Parse url and add to the list
				u, err := url.Parse(attr.Val)
				if err != nil {
					continue
				}

				var resolved string
				if u.IsAbs() {
					resolved = u.String()
				} else {
					resolved = baseURL.ResolveReference(u).String()
				}

				resolved, err = utils.NormalizeURL(resolved)
				if err != nil {
					continue
				}

				imageDetails["src"] = resolved
			} else if attr.Key == "alt" {
				imageDetails["alt"] = attr.Val
			}
		}

		if imgURL, hasSrc := imageDetails["src"]; hasSrc && imgURL != "" {
			imagesMap[imgURL] = imageDetails
		}
	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		traverse(c, baseURL, linksSet, imagesMap, directives)
	}
}