package web

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return false
	}
	return true
}

func decodeOptionalJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return true
		}
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return false
	}
	return true
}

func queryBool(r *http.Request, key string) bool {
	value := strings.TrimSpace(strings.ToLower(r.URL.Query().Get(key)))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func parseOptionalRFC3339(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: timestamp must be RFC3339", app.ErrInvalidInput)
	}
	return parsed, nil
}

func queryInt(r *http.Request, key string) int {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func validateWebRelativePath(relativePath string) error {
	trimmed := strings.TrimSpace(relativePath)
	if trimmed == "" {
		return nil
	}
	if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, `\`) || strings.HasPrefix(trimmed, "~") {
		return fmt.Errorf("%w: relative_path must be root-relative", app.ErrInvalidInput)
	}
	firstSegment := trimmed
	if slash := strings.IndexAny(firstSegment, `/\`); slash >= 0 {
		firstSegment = firstSegment[:slash]
	}
	if strings.Contains(firstSegment, ":") {
		return fmt.Errorf("%w: relative_path must be root-relative", app.ErrInvalidInput)
	}
	return nil
}

func sourceEventID(event *app.LedgerEvent) string {
	if event == nil {
		return ""
	}
	return event.EventID
}

func localPathLocatorKind(snapshot app.SourceSnapshot) string {
	var locators []app.LocalPathLocator
	if err := json.Unmarshal(snapshot.Locators, &locators); err != nil {
		return ""
	}
	for _, locator := range locators {
		if locatorType(locator.LocatorType, locator.Kind) == app.SourceLocatorTypeLocalPath {
			return strings.TrimSpace(locator.PathKind)
		}
	}
	return ""
}

func jsonWrite(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return true
	}
	contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil && strings.EqualFold(mediaType, "multipart/form-data") && strings.HasSuffix(r.URL.Path, "/sources/upload") {
		return true
	}
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(value)
}

func writeAppError(w http.ResponseWriter, err error) {
	if confluenceErr, ok := app.ConfluenceErrorDetails(err); ok {
		writeConfluenceError(w, confluenceErr)
		return
	}
	if errors.Is(err, app.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, appErrorMessage(err))
		return
	}
	if errors.Is(err, app.ErrConflict) {
		writeError(w, http.StatusConflict, appErrorMessage(err))
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}

func writeConfluenceError(w http.ResponseWriter, err *app.ConfluenceError) {
	status := err.HTTPStatus
	if status <= 0 {
		status = http.StatusBadRequest
	}
	payload := map[string]any{
		"status":   status,
		"message":  err.Error(),
		"category": err.Category,
		"code":     err.Code,
	}
	if strings.TrimSpace(err.RetryAfter) != "" {
		payload["retry_after"] = strings.TrimSpace(err.RetryAfter)
	}
	if strings.TrimSpace(err.Operation) != "" {
		payload["operation"] = strings.TrimSpace(err.Operation)
	}
	writeJSON(w, status, map[string]any{"error": payload})
}

func appErrorMessage(err error) string {
	message := err.Error()
	for _, base := range []error{app.ErrInvalidInput, app.ErrConflict} {
		prefix := base.Error() + ": "
		if strings.HasPrefix(message, prefix) {
			message = strings.TrimSpace(strings.TrimPrefix(message, prefix))
			break
		}
	}
	if message == "" {
		return app.ErrInvalidInput.Error()
	}
	return message
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"status":  status,
			"message": message,
		},
	})
}

func mustJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func newID(prefix string) string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s_%s_%s", prefix, time.Now().UTC().Format("20060102150405"), hex.EncodeToString(b[:]))
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func trimStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func headTailExcerpt(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	headLimit := limit / 2
	tailLimit := limit - headLimit
	return value[:headLimit] + "\n[truncated middle]\n" + value[len(value)-tailLimit:]
}

func sameOriginWrite(r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return true
	}
	if strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")) == "cross-site" {
		return false
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Scheme, requestScheme(r)) && strings.EqualFold(parsed.Host, r.Host)
}

func requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		return strings.ToLower(forwarded)
	}
	return "http"
}
