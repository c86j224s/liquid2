package web

func (store *liveTurnStore) subscribe(missionID, userEventID string) (liveTurnSnapshot, <-chan liveTurnSnapshot, func(), bool) {
	store.mu.Lock()
	defer store.mu.Unlock()
	entry := store.entryLocked(liveTurnKey{missionID: missionID, userEventID: userEventID})
	if entry == nil {
		return liveTurnSnapshot{}, nil, func() {}, false
	}
	ch := make(chan liveTurnSnapshot, 8)
	entry.subscribers[ch] = struct{}{}
	unsubscribe := func() {
		store.mu.Lock()
		defer store.mu.Unlock()
		if current := store.entryLocked(liveTurnKey{missionID: missionID, userEventID: userEventID}); current != nil {
			delete(current.subscribers, ch)
		}
	}
	return entry.snapshot, ch, unsubscribe, true
}

func (store *liveTurnStore) withEntry(missionID, userEventID string, update func(*liveTurnEntry) (liveTurnSnapshot, bool, bool)) {
	key := liveTurnKey{missionID: missionID, userEventID: userEventID}
	store.mu.Lock()
	entry := store.entryLocked(key)
	if entry == nil {
		store.mu.Unlock()
		return
	}
	snapshot, publish, terminal := update(entry)
	subscribers := make([]chan liveTurnSnapshot, 0, len(entry.subscribers))
	for ch := range entry.subscribers {
		subscribers = append(subscribers, ch)
	}
	if terminal {
		delete(store.turns, key)
	}
	store.mu.Unlock()

	if !publish {
		return
	}
	for _, ch := range subscribers {
		publishLiveTurnSnapshot(ch, snapshot, terminal)
		if terminal {
			close(ch)
		}
	}
}

func publishLiveTurnSnapshot(ch chan liveTurnSnapshot, snapshot liveTurnSnapshot, terminal bool) {
	select {
	case ch <- snapshot:
		return
	default:
	}
	if !terminal {
		return
	}
	select {
	case <-ch:
	default:
	}
	ch <- snapshot
}

func (store *liveTurnStore) entryLocked(key liveTurnKey) *liveTurnEntry {
	if store.turns == nil {
		return nil
	}
	return store.turns[key]
}
