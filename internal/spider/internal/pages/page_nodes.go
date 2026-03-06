package pages

// PageNode represents a node in the web link graph, holding a page URL
// and the set of URLs it links to (outlinks) or is linked from (backlinks).
// The NormalizedLinkURLs map uses struct{} values as a memory-efficient set.
type PageNode struct {
	NormalizedURL      string              // Canonical URL of this page
	NormalizedLinkURLs map[string]struct{} // Set of linked URLs (no duplicate entries)
}

// GetLinks returns all linked URLs as a string slice.
func (b *PageNode) GetLinks() []string {
    var links []string
    for link := range b.NormalizedLinkURLs {
        links = append(links, link)
    }

    return links
}

// CreatePageNode constructs a new PageNode with an initialized empty link set.
func CreatePageNode(normalizedURL string) *PageNode {
    return &PageNode {
        NormalizedURL:  normalizedURL,
        NormalizedLinkURLs:   make(map[string]struct{}),
    }
}

// AppendLink adds a normalized URL to this node's link set.
// Initializes the set if nil. Duplicate links are naturally ignored.
func (b *PageNode) AppendLink(newNormalizedLink string) {
    // Check if NormalizedLinkURLs has been initialized before
    if b.NormalizedLinkURLs == nil {
        b.NormalizedLinkURLs = make(map[string]struct{})
    }

    b.NormalizedLinkURLs[newNormalizedLink] = struct{}{}
}
