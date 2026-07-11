package app

import "context"

const demoSeedTime int64 = 1760000000000

// WithDemoSeed populates local in-memory review data for explicit demo runs.
func WithDemoSeed() Option {
	return func(config *serviceConfig) {
		config.seeds = append(config.seeds, seedDemoState)
	}
}

func (s *Service) SeedDemo(ctx context.Context) error {
	return s.repo.Update(ctx, func(tx RepositoryTx) error {
		seedDemoState(tx)
		return nil
	})
}

func seedDemoState(tx RepositoryTx) {
	inboxID := ensureDefaultDocumentFolder(tx)
	researchID := "folder_demo_research"
	feedsID := ensureFeedsFolder(tx)
	tx.PutFolder(demoFolder(inboxID, nil, "Inbox", folderSystemRolePtr(FolderSystemRoleInbox), 10))
	tx.PutFolder(demoFolder(researchID, &inboxID, "Research", nil, 20))
	tx.PutFolder(demoFolder(feedsID, nil, "Feeds", folderSystemRolePtr(FolderSystemRoleFeeds), 30))

	addDemoTag(tx, "tag_demo_sqlite", "SQLite", "sqlite")
	addDemoTag(tx, "tag_demo_go", "Go", "go")
	addDemoTag(tx, "tag_demo_flutter", "Flutter", "flutter")
	addDemoTag(tx, "tag_demo_backend", "Backend", "backend")

	readAt := demoSeedTime + 1800000
	tx.PutDocument(documentRecord{
		meta: DocumentMetadata{
			ID: "doc_demo_sqlite", Title: "SQLite에 1MB 문서 저장 전략",
			Kind: DocumentKindBookmark, FolderID: &researchID,
			SourceURL:    strptr("https://example.com/sqlite-doc-store"),
			CanonicalURL: strptr("https://example.com/sqlite-doc-store"),
			Language:     strptr("ko"), Status: DocumentStatusUnread, Rating: intptr(5),
			CreatedAt: demoSeedTime, UpdatedAt: demoSeedTime,
		},
		contents: []DocumentContent{demoContent(
			"content_demo_sqlite", "markdown", "ko",
			"SQLite에 1MB 이하 문서를 저장하는 것은 개인 문서 저장소 MVP에서는 충분히 현실적인 선택입니다.\n\n메타데이터, 태그, 읽음 상태, 평점은 관계형 테이블로 관리하고 원문은 content variant로 분리하면 됩니다.",
		)},
		tagIDs: []string{"tag_demo_sqlite", "tag_demo_backend"},
	})
	tx.PutDocument(documentRecord{
		meta: DocumentMetadata{
			ID: "doc_demo_go", Title: "Go 서비스 경계와 actor 상태 메모",
			Kind: "scraped_article", FolderID: &researchID,
			SourceURL: strptr("https://example.com/go-service-state"),
			Language:  strptr("ko"), Status: DocumentStatusRead, Rating: intptr(4),
			CreatedAt: demoSeedTime + 600000, UpdatedAt: demoSeedTime + 600000,
			ReadAt: &readAt,
		},
		contents: []DocumentContent{demoContent(
			"content_demo_go", "markdown", "ko",
			"서비스 상태를 한 goroutine 안에 모으면 초기 구현은 단순해집니다.\n\n장기적으로는 shutdown, panic recovery, persistence 경계를 분리해야 합니다.",
		)},
		tagIDs: []string{"tag_demo_go", "tag_demo_backend"},
	})
	tx.PutDocument(documentRecord{
		meta: DocumentMetadata{
			ID: "doc_demo_flutter", Title: "Flutter shell 리뷰 체크리스트",
			Kind: "rss_item", FolderID: &feedsID,
			SourceURL: strptr("https://example.com/flutter-shell-review"),
			Language:  strptr("ko"), Status: DocumentStatusUnread,
			CreatedAt: demoSeedTime + 1200000, UpdatedAt: demoSeedTime + 1200000,
		},
		contents: []DocumentContent{demoContent(
			"content_demo_flutter", "text", "ko",
			"목록, 상세, 노트, 태그, 평점, 읽음 상태가 API 계약을 통해 동작하는지 확인합니다.",
		)},
		tagIDs: []string{"tag_demo_flutter"},
	})
	tx.PutNote(DocumentNote{
		ID: "note_demo_sqlite", DocumentID: "doc_demo_sqlite",
		Body:   "PostgreSQL 이전 가능성을 고려해 BLOB 저장 정책은 인터페이스 뒤에 둔다.",
		Format: "text", CreatedAt: demoSeedTime + 300000,
		UpdatedAt: demoSeedTime + 300000,
	})
}

func demoFolder(id string, parentID *string, name string, systemRole *string, sortOrder int) Folder {
	return Folder{
		ID: id, ParentID: cloneString(parentID), Name: name,
		SystemRole: cloneString(systemRole), SortOrder: sortOrder,
		CreatedAt: demoSeedTime, UpdatedAt: demoSeedTime, Children: []Folder{},
	}
}

func addDemoTag(tx RepositoryTx, id string, name string, slug string) {
	tx.PutTag(Tag{ID: id, Name: name, Slug: slug, CreatedAt: demoSeedTime})
}

func demoContent(id string, format string, language string, content string) DocumentContent {
	return DocumentContent{
		ID: id, Role: "original", Format: format, Language: strptr(language),
		Content: content,
	}
}

func strptr(value string) *string {
	return &value
}

func intptr(value int) *int {
	return &value
}
