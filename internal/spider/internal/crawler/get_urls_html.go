package crawler

import ( 
	"strings"
	"net/url"
	"regexp"
	"golang.org/x/net/html"
	"spider/internal/utils"
)

// getURLsFromHTML parses an HTML document and extracts all hyperlinks and images.
//
// Returns:
//   - links: deduplicated slice of absolute URLs extracted from <a href> tags
//   - imagesMap: map of normalized image source URLs to their attributes
//     (keys: image URL, values: map with "src" and optionally "alt")
//   - err: non-nil if the base URL or HTML could not be parsed
//
// Relative URLs are resolved against rawURL. Malformed and non-ASCII URLs
// are silently skipped.
func getURLsFromHTML(htmlBody string, rawURL string) ([]string, map[string]map[string]string, error) {
    baseURL, err := url.Parse(rawURL)
    if err != nil {
        // Couldn't parse baseURL
        return nil, nil, err
    }

    node, err := html.Parse(strings.NewReader(htmlBody))
    if err != nil {
        return nil, nil, err
    }

    linksSet := make(map[string]struct{}) //sets in go
    imagesMap := make(map[string]map[string]string)

    traverse(node, baseURL, linksSet, imagesMap)

    // Convert set to slice
    links := make([]string, 0, len(linksSet))
    for link := range linksSet {
        links = append(links, link)
    }

    return links, imagesMap, nil
}

// nonASCIIRegex matches any character outside the printable ASCII range.
// Used to filter out URLs containing non-ASCII characters.
var nonASCIIRegex = regexp.MustCompile(`[^\x20-\x7E]`)

// traverse performs a depth-first walk of the HTML node tree, collecting
// <a href> links into linksSet and <img src/alt> data into imagesMap.
// Relative URLs are resolved against baseURL.
func traverse(node *html.Node, baseURL *url.URL, linksSet map[string]struct{}, imagesMap map[string]map[string]string) {
	if node == nil {
		return
	}

	//detects if the node is an anchor tag and extracts the href attribute
	if node.Type == html.ElementNode && node.Data == "a" {
		for _, attr := range node.Attr {
			if attr.Key == "href" {
                rawHref := attr.Val

                // Skip malformed URLS
                if strings.ContainsAny(rawHref, " <>\"") {
                    continue
                }

                // Skip non-ASCII urls
                if nonASCIIRegex.MatchString(rawHref) {
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

                // Append to list
                linksSet[resolved] = struct{}{}
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
                    // Could not normalize image URL
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
        traverse(c, baseURL, linksSet, imagesMap)
    }
}