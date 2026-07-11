package app

func (reader memoryReader) Feed(id string) (Feed, bool) {
	feed, ok := reader.state.feeds[id]
	if !ok {
		return Feed{}, false
	}
	return cloneFeed(*feed), true
}

func (reader memoryReader) FeedByURL(url string) (Feed, bool) {
	id, ok := reader.state.feedURLs[url]
	if !ok {
		return Feed{}, false
	}
	return reader.Feed(id)
}

func (reader memoryReader) Feeds() []Feed {
	feeds := make([]Feed, 0, len(reader.state.feeds))
	for _, feed := range reader.state.feeds {
		feeds = append(feeds, cloneFeed(*feed))
	}
	return feeds
}

func (reader memoryReader) FeedItemByDocumentID(documentID string) (FeedItem, bool) {
	for _, feedItems := range reader.state.items {
		for _, item := range feedItems {
			if item.DocumentID == documentID {
				return cloneFeedItem(*item), true
			}
		}
	}
	return FeedItem{}, false
}

func (reader memoryReader) FeedItems(feedID string) []FeedItem {
	items := make([]FeedItem, 0, len(reader.state.items[feedID]))
	for _, item := range reader.state.items[feedID] {
		items = append(items, cloneFeedItem(*item))
	}
	return items
}

func (reader memoryReader) Job(id string) (Job, bool) {
	job, ok := reader.state.jobs[id]
	if !ok {
		return Job{}, false
	}
	return cloneJob(*job), true
}

func (reader memoryReader) Jobs(filters JobFilters) []Job {
	jobs := make([]Job, 0, len(reader.state.jobs))
	for _, job := range reader.state.jobs {
		if matchesJobFilters(*job, filters) {
			jobs = append(jobs, cloneJob(*job))
		}
	}
	sortJobs(jobs)
	return limitJobs(jobs, filters.Limit)
}

func (tx memoryTx) PutFeed(feed Feed) {
	cloned := cloneFeed(feed)
	if existing, ok := tx.state.feeds[feed.ID]; ok {
		delete(tx.state.feedURLs, existing.URL)
	}
	tx.state.feeds[feed.ID] = &cloned
	tx.state.feedURLs[feed.URL] = feed.ID
}

func (tx memoryTx) DeleteFeed(id string) {
	if existing, ok := tx.state.feeds[id]; ok {
		delete(tx.state.feedURLs, existing.URL)
	}
	delete(tx.state.feeds, id)
	delete(tx.state.items, id)
}

func (tx memoryTx) PutFeedItem(item FeedItem) {
	cloned := cloneFeedItem(item)
	if tx.state.items[item.FeedID] == nil {
		tx.state.items[item.FeedID] = map[string]*FeedItem{}
	}
	for id, existing := range tx.state.items[item.FeedID] {
		if id != item.ID && duplicateFeedItem(*existing, item) {
			panic(memoryAbort{err: conflict("feed item already exists")})
		}
	}
	tx.state.items[item.FeedID][item.ID] = &cloned
}

func (tx memoryTx) PutJob(job Job) {
	cloned := cloneJob(job)
	tx.state.jobs[job.ID] = &cloned
}

func duplicateFeedItem(existing FeedItem, next FeedItem) bool {
	return sameSetString(existing.GUID, next.GUID) ||
		sameSetString(existing.CanonicalURL, next.CanonicalURL) ||
		sameSetString(existing.ContentHash, next.ContentHash) ||
		existing.URL == next.URL
}

func sameSetString(left *string, right *string) bool {
	return left != nil && right != nil && *left == *right
}
