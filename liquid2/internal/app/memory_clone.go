package app

func cloneRepositoryState(state *repositoryState) *repositoryState {
	cloned := newEmptyRepositoryState(state.now)
	cloned.next = state.next
	cloned.settings = cloneAppSettings(state.settings)
	cloneDocumentState(cloned, state)
	cloneFolderTagState(cloned, state)
	cloneFeedJobState(cloned, state)
	return cloned
}

func cloneDocumentState(cloned *repositoryState, state *repositoryState) {
	for id, record := range state.docs {
		copied := cloneDocumentRecord(*record)
		cloned.docs[id] = &copied
	}
	for documentID, versions := range state.versions {
		cloned.versions[documentID] = cloneDocumentVersionPointers(versions)
	}
	for documentID, notes := range state.notes {
		cloned.notes[documentID] = map[string]*DocumentNote{}
		for noteID, note := range notes {
			copied := cloneNote(*note)
			cloned.notes[documentID][noteID] = &copied
		}
	}
}

func cloneFolderTagState(cloned *repositoryState, state *repositoryState) {
	for id, folder := range state.folders {
		copied := cloneFolder(*folder)
		cloned.folders[id] = &copied
	}
	for id, tag := range state.tags {
		copied := *tag
		cloned.tags[id] = &copied
	}
	for slug, id := range state.tagSlugs {
		cloned.tagSlugs[slug] = id
	}
}

func cloneFeedJobState(cloned *repositoryState, state *repositoryState) {
	for id, feed := range state.feeds {
		copied := cloneFeed(*feed)
		cloned.feeds[id] = &copied
	}
	for url, id := range state.feedURLs {
		cloned.feedURLs[url] = id
	}
	for feedID, items := range state.items {
		cloned.items[feedID] = map[string]*FeedItem{}
		for itemID, item := range items {
			copied := cloneFeedItem(*item)
			cloned.items[feedID][itemID] = &copied
		}
	}
	for id, job := range state.jobs {
		copied := cloneJob(*job)
		cloned.jobs[id] = &copied
	}
}
