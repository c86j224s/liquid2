package app

type feedItemSet struct {
	guids         map[string]struct{}
	canonicalURLs map[string]struct{}
	contentHashes map[string]struct{}
	urls          map[string]struct{}
}

func newFeedItemSet(items []FeedItem) *feedItemSet {
	set := &feedItemSet{
		guids:         map[string]struct{}{},
		canonicalURLs: map[string]struct{}{},
		contentHashes: map[string]struct{}{},
		urls:          map[string]struct{}{},
	}
	for _, item := range items {
		set.addFeedItem(item)
	}
	return set
}

func (set *feedItemSet) has(item normalizedFeedItem) bool {
	return hasString(set.guids, item.guid) ||
		hasString(set.canonicalURLs, item.canonicalURL) ||
		hasString(set.contentHashes, item.contentHash) ||
		hasValue(set.urls, item.url)
}

func (set *feedItemSet) add(item normalizedFeedItem) {
	addString(set.guids, item.guid)
	addString(set.canonicalURLs, item.canonicalURL)
	addString(set.contentHashes, item.contentHash)
	addValue(set.urls, item.url)
}

func (set *feedItemSet) addFeedItem(item FeedItem) {
	addString(set.guids, item.GUID)
	addString(set.canonicalURLs, item.CanonicalURL)
	addString(set.contentHashes, item.ContentHash)
	addValue(set.urls, item.URL)
}

func hasString(values map[string]struct{}, value *string) bool {
	if value == nil {
		return false
	}
	return hasValue(values, *value)
}

func hasValue(values map[string]struct{}, value string) bool {
	_, ok := values[value]
	return ok
}

func addString(values map[string]struct{}, value *string) {
	if value != nil {
		addValue(values, *value)
	}
}

func addValue(values map[string]struct{}, value string) {
	if value != "" {
		values[value] = struct{}{}
	}
}
