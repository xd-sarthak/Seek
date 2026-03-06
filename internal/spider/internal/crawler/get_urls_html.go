package crawler

import ( 
	"strings"
	"net/url"
	"regexp"
	"golang.org/x/net/html"
	"spider/internal/utils"
)

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

var nonASCIIRegex = regexp.MustCompile(`[^\x20-\x7E]`)

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

        if len(imageDetails) > 0 {
            imgURL := imageDetails["src"]
            imagesMap[imgURL] = imageDetails
        }
	}

	    for c := node.FirstChild; c != nil; c = c.NextSibling {
        traverse(c, baseURL, linksSet, imagesMap)
    }
}