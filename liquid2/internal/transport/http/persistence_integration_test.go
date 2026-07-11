package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
	sqlitestore "github.com/c86j224s/liquid2/internal/storage/sqlite"
)

func TestSQLiteHTTPPersistenceAcrossRestart(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "liquid2.db")
	router, closeRouter := sqliteHTTPRouter(t, ctx, dbPath)

	folder := postJSON[app.Folder](t, router, "/api/v1/folders", `{"name":"Research","sortOrder":10}`)
	tag := postJSON[app.Tag](t, router, "/api/v1/tags", `{"name":"SQLite"}`)
	bookmark := postJSON[app.DocumentDetail](t, router, "/api/v1/documents/bookmark",
		`{"url":"https://example.com/bookmark","title":"Saved Link","folderId":"`+folder.ID+`","tagIds":["`+tag.ID+`"]}`)
	scrape := postJSON[app.DocumentDetail](t, router, "/api/v1/documents/scrape",
		`{"url":"https://example.com/scrape","folderId":"`+folder.ID+`","tagIds":["`+tag.ID+`"]}`)
	note := postJSON[app.DocumentNote](t, router, "/api/v1/documents/"+bookmark.Document.ID+"/notes",
		`{"body":"Read later","format":"text"}`)
	upload := uploadTextDocument(t, router, "upload.txt", "Uploaded body")
	closeRouter()

	router, closeRouter = sqliteHTTPRouter(t, ctx, dbPath)
	defer closeRouter()

	assertDocumentPersisted(t, router, bookmark.Document.ID, "Saved Link", folder.ID, tag.ID)
	assertDocumentPersisted(t, router, scrape.Document.ID, "Fetched title", folder.ID, tag.ID)
	assertDocumentPersisted(t, router, upload.Document.ID, "Uploaded note", "", "")
	assertNotePersisted(t, router, bookmark.Document.ID, note.ID, "Read later")
	assertCollectionContains(t, router, "/api/v1/folders", `"name":"Research"`)
	assertCollectionContains(t, router, "/api/v1/tags", `"slug":"sqlite"`)
}

func sqliteHTTPRouter(t *testing.T, ctx context.Context, dbPath string) (http.Handler, func()) {
	t.Helper()
	store, err := sqlitestore.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	if err := store.Migrate(ctx); err != nil {
		_ = store.Close()
		t.Fatalf("migrate sqlite store: %v", err)
	}
	repo := app.NewSQLiteRepository(store)
	service := app.NewService(app.WithRepository(repo))
	router := ingestionTestRouter(service)
	return router, func() {
		if err := service.Close(); err != nil {
			t.Fatalf("close service: %v", err)
		}
		if err := store.Close(); err != nil {
			t.Fatalf("close sqlite store: %v", err)
		}
	}
}

func postJSON[T any](t *testing.T, router http.Handler, path string, body string) T {
	t.Helper()
	response := serveJSON(router, http.MethodPost, path, body)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201 from %s, got %d: %s", path, response.Code, response.Body.String())
	}
	return decodeBody[T](t, response)
}

func uploadTextDocument(t *testing.T, router http.Handler, filename string, content string) app.DocumentDetail {
	t.Helper()
	body, contentType := multipartBody(t, filename, "text/plain", content)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents/upload", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected upload status 201, got %d: %s", response.Code, response.Body.String())
	}
	return decodeBody[app.DocumentDetail](t, response)
}

func assertDocumentPersisted(t *testing.T, router http.Handler, id string, title string, folderID string, tagID string) {
	t.Helper()
	response := serveJSON(router, http.MethodGet, "/api/v1/documents/"+id, "")
	if response.Code != http.StatusOK {
		t.Fatalf("expected document status 200, got %d: %s", response.Code, response.Body.String())
	}
	detail := decodeBody[app.DocumentDetail](t, response)
	if detail.Document.Title != title {
		t.Fatalf("expected title %q, got %#v", title, detail.Document)
	}
	if folderID != "" && (detail.Document.FolderID == nil || *detail.Document.FolderID != folderID) {
		t.Fatalf("expected folder %s, got %#v", folderID, detail.Document.FolderID)
	}
	if tagID != "" && (len(detail.Tags) != 1 || detail.Tags[0].ID != tagID) {
		t.Fatalf("expected tag %s, got %#v", tagID, detail.Tags)
	}
}

func assertNotePersisted(t *testing.T, router http.Handler, documentID string, noteID string, body string) {
	t.Helper()
	response := serveJSON(router, http.MethodGet, "/api/v1/documents/"+documentID+"/notes", "")
	if response.Code != http.StatusOK {
		t.Fatalf("expected notes status 200, got %d: %s", response.Code, response.Body.String())
	}
	notes := decodeBody[app.NoteList](t, response)
	if len(notes.Items) != 1 || notes.Items[0].ID != noteID || notes.Items[0].Body != body {
		t.Fatalf("unexpected notes %#v", notes.Items)
	}
}

func assertCollectionContains(t *testing.T, router http.Handler, path string, fragment string) {
	t.Helper()
	response := serveJSON(router, http.MethodGet, path, "")
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200 from %s, got %d: %s", path, response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), fragment) {
		t.Fatalf("expected %s to contain %s, got %s", path, fragment, response.Body.String())
	}
}

func decodeBody[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode response: %v: %s", err, response.Body.String())
	}
	return value
}
