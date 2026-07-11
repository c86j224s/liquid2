package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	htmlpkg "html"
	"image"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidates"
	"github.com/c86j224s/liquid2/plasma/internal/sourceingest"
	"github.com/c86j224s/liquid2/plasma/internal/sources/pdftext"
)

func (server *Server) recordSourceSnapshotFailure(ctx context.Context, missionID string, sourceKind string, normalizedURL string, cause error) {
	_, _ = server.service.AppendEvent(ctx, sourceingest.BuildSourceSnapshotFailureAppendRequest(sourceingest.SourceSnapshotFailureAppendRequest{
		EventID:    newID("evt"),
		MissionID:  missionID,
		SourceKind: sourceKind,
		URL:        normalizedURL,
		Message:    appErrorMessage(cause),
		Producer:   app.Producer{Type: "user", ID: "plasma-ui"},
	}))
}

type fetchedURLSource struct {
	Content           []byte
	MediaType         string
	Title             string
	ExternalVersion   string
	ExternalUpdatedAt time.Time
	ByteSize          int64
	PageCount         int
	TextLength        int
	TextLengthKnown   bool
}

type fetchedMediaSource struct {
	Content           []byte
	MediaType         string
	MediaKind         string
	Title             string
	ExternalVersion   string
	ExternalUpdatedAt time.Time
	ByteSize          int64
	Width             int
	Height            int
}

type fetchedPDFSource struct {
	Content           []byte
	MediaType         string
	Title             string
	ExternalVersion   string
	ExternalUpdatedAt time.Time
	ByteSize          int64
	PageCount         int
	TextLength        int
	TextLengthKnown   bool
}

func normalizedHTTPURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%w: source URL is required", app.ErrInvalidInput)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("%w: source URL must be absolute", app.ErrInvalidInput)
	}
	if parsed.User != nil {
		return "", fmt.Errorf("%w: source URL must not include credentials", app.ErrInvalidInput)
	}
	parsed.Fragment = ""
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		parsed.Scheme = strings.ToLower(parsed.Scheme)
	default:
		return "", fmt.Errorf("%w: source URL must use http or https", app.ErrInvalidInput)
	}
	parsed.Host = strings.ToLower(parsed.Host)
	return parsed.String(), nil
}

type confluencePageURLTarget struct {
	RawURL  string
	SiteURL string
	CloudID string
	PageID  string
}

func confluencePageIDFromURL(parsed *url.URL) string {
	if parsed == nil {
		return ""
	}
	if pageID := strings.TrimSpace(parsed.Query().Get("pageId")); isConfluencePageID(pageID) {
		return pageID
	}
	segments := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	for index, segment := range segments {
		unescaped, err := url.PathUnescape(segment)
		if err != nil {
			continue
		}
		if unescaped == "pages" && index+1 < len(segments) {
			next, err := url.PathUnescape(segments[index+1])
			if err == nil && isConfluencePageID(next) {
				return next
			}
		}
		if unescaped == "edit-v2" && index+1 < len(segments) {
			next, err := url.PathUnescape(segments[index+1])
			if err == nil && isConfluencePageID(next) {
				return next
			}
		}
	}
	return ""
}

func isConfluencePageID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func fetchURLSource(ctx context.Context, rawURL string) (fetchedURLSource, error) {
	return fetchURLSourceWithClient(ctx, rawURL, secureURLSourceHTTPClient())
}

func fetchMediaSource(ctx context.Context, rawURL string) (fetchedMediaSource, error) {
	return fetchMediaSourceWithClient(ctx, rawURL, secureURLSourceHTTPClient())
}

func fetchPDFSource(ctx context.Context, rawURL string) (fetchedPDFSource, error) {
	return fetchPDFSourceWithClient(ctx, rawURL, secureURLSourceHTTPClient())
}

const (
	maxURLSourceBytes              = 20 << 20
	maxImageMediaSourceBytes       = 10 << 20
	maxPDFSourceBytes              = 100 << 20
	urlSourceDialTimeout           = 15 * time.Second
	urlSourceTLSHandshakeTimeout   = 15 * time.Second
	urlSourceResponseHeaderTimeout = 45 * time.Second
	urlSourceFetchTimeout          = 60 * time.Second
)

func fetchURLSourceWithClient(ctx context.Context, rawURL string, client *http.Client) (fetchedURLSource, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fetchedURLSource{}, fmt.Errorf("%w: invalid source URL", app.ErrInvalidInput)
	}
	req.Header.Set("Accept", "application/pdf,text/html,text/plain,application/json,application/xml,text/*,*/*;q=0.1")
	req.Header.Set("Accept-Language", "ko,en-US;q=0.9,en;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; PlasmaSourceFetcher/0.1; +https://github.com/c86j224s/liquid2)")
	resp, err := client.Do(req)
	if err != nil {
		return fetchedURLSource{}, fmt.Errorf("%w: %s", app.ErrInvalidInput, sourceURLFetchFailureMessage(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fetchedURLSource{}, fmt.Errorf("%w: %s", app.ErrInvalidInput, sourceURLHTTPStatusMessage(resp.StatusCode))
	}
	declaredType := responseHeaderMediaType(resp.Header.Get("Content-Type"))
	pdfExpected := pdftext.IsPDFMediaType(declaredType) || strings.HasSuffix(strings.ToLower(resp.Request.URL.Path), ".pdf")
	limit := maxURLSourceBytes
	limitLabel := "20 MiB"
	if pdfExpected {
		limit = maxPDFSourceBytes
		limitLabel = "100 MiB"
	}
	if resp.ContentLength > int64(limit) {
		return fetchedURLSource{}, fmt.Errorf("%w: source URL response is larger than %s", app.ErrInvalidInput, limitLabel)
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, int64(limit)+1))
	if err != nil {
		return fetchedURLSource{}, err
	}
	if len(content) == 0 {
		return fetchedURLSource{}, fmt.Errorf("%w: source URL returned empty content", app.ErrInvalidInput)
	}
	if len(content) > limit {
		return fetchedURLSource{}, fmt.Errorf("%w: source URL response is larger than %s", app.ErrInvalidInput, limitLabel)
	}
	mediaType := responseMediaType(resp.Header.Get("Content-Type"), content)
	if pdftext.IsPDFMediaType(mediaType) || pdftext.IsPDFBytes(content) {
		info, err := pdftext.Inspect(content)
		if err != nil {
			return fetchedURLSource{}, fmt.Errorf("%w: PDF inspection failed: %v", app.ErrInvalidInput, err)
		}
		return fetchedURLSource{
			Content:           content,
			MediaType:         pdftext.MediaType,
			Title:             urlTitleFromPath(resp.Request.URL),
			ExternalVersion:   responseExternalVersion(resp.Header),
			ExternalUpdatedAt: responseLastModified(resp.Header),
			ByteSize:          int64(len(content)),
			PageCount:         info.PageCount,
			TextLengthKnown:   false,
		}, nil
	}
	if !isTextualMediaType(mediaType) {
		return fetchedURLSource{}, fmt.Errorf("%w: source URL content type %q is not supported", app.ErrInvalidInput, mediaType)
	}
	if looksLikeAtlassianAuthWall(resp.Request.URL, content, mediaType) {
		return fetchedURLSource{}, fmt.Errorf("%w: URL 원문이 Atlassian 로그인 화면으로 확인되었습니다. Confluence 자료는 Settings에서 API token 연결을 만든 뒤 Sources의 Confluence에서 page를 검색해 추가하세요.", app.ErrInvalidInput)
	}
	var updatedAt time.Time
	if lastModified := strings.TrimSpace(resp.Header.Get("Last-Modified")); lastModified != "" {
		if parsed, err := http.ParseTime(lastModified); err == nil {
			updatedAt = parsed
		}
	}
	return fetchedURLSource{
		Content:           content,
		MediaType:         mediaType,
		Title:             htmlTitle(content, mediaType),
		ExternalVersion:   responseExternalVersion(resp.Header),
		ExternalUpdatedAt: updatedAt,
		ByteSize:          int64(len(content)),
		TextLength:        len(content),
		TextLengthKnown:   true,
	}, nil
}

func urlTitleFromPath(parsed *url.URL) string {
	if parsed == nil {
		return ""
	}
	path := strings.TrimRight(parsed.Path, "/")
	if path == "" {
		return parsed.String()
	}
	idx := strings.LastIndex(path, "/")
	if idx >= 0 {
		return path[idx+1:]
	}
	return path
}

func looksLikeAtlassianAuthWall(finalURL *url.URL, content []byte, mediaType string) bool {
	if finalURL == nil || !strings.Contains(mediaType, "html") {
		return false
	}
	host := strings.ToLower(finalURL.Hostname())
	if host != "atlassian.com" && !strings.HasSuffix(host, ".atlassian.com") && host != "atlassian.net" && !strings.HasSuffix(host, ".atlassian.net") {
		return false
	}
	path := strings.ToLower(finalURL.EscapedPath())
	if strings.Contains(path, "login") {
		return true
	}
	body := strings.ToLower(string(content))
	return strings.Contains(body, "log in with atlassian") ||
		strings.Contains(body, "atlassian account") && strings.Contains(body, "login") ||
		strings.Contains(body, "id.atlassian.com/login")
}

func fetchMediaSourceWithClient(ctx context.Context, rawURL string, client *http.Client) (fetchedMediaSource, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fetchedMediaSource{}, fmt.Errorf("%w: invalid media source URL", app.ErrInvalidInput)
	}
	req.Header.Set("Accept", "image/png,image/jpeg,image/gif,audio/*,video/*,*/*;q=0.1")
	req.Header.Set("Accept-Language", "ko,en-US;q=0.9,en;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; PlasmaMediaFetcher/0.1; +https://github.com/c86j224s/liquid2)")
	resp, err := client.Do(req)
	if err != nil {
		return fetchedMediaSource{}, fmt.Errorf("%w: %s", app.ErrInvalidInput, sourceURLFetchFailureMessage(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fetchedMediaSource{}, fmt.Errorf("%w: %s", app.ErrInvalidInput, sourceURLHTTPStatusMessage(resp.StatusCode))
	}
	declaredType := responseHeaderMediaType(resp.Header.Get("Content-Type"))
	declaredKind := mediaKindForType(declaredType)
	if declaredKind == app.MediaKindAudio || declaredKind == app.MediaKindVideo {
		return fetchedMediaSource{
			MediaType:         declaredType,
			MediaKind:         declaredKind,
			ExternalVersion:   responseExternalVersion(resp.Header),
			ExternalUpdatedAt: responseLastModified(resp.Header),
			ByteSize:          normalizedContentLength(resp.ContentLength),
		}, nil
	}
	if declaredKind == app.MediaKindImage && resp.ContentLength > maxImageMediaSourceBytes {
		return fetchedMediaSource{}, fmt.Errorf("%w: image source response is larger than 10 MiB", app.ErrInvalidInput)
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, maxImageMediaSourceBytes+1))
	if err != nil {
		return fetchedMediaSource{}, err
	}
	if len(content) == 0 {
		return fetchedMediaSource{}, fmt.Errorf("%w: media source returned empty content", app.ErrInvalidInput)
	}
	if len(content) > maxImageMediaSourceBytes {
		return fetchedMediaSource{}, fmt.Errorf("%w: image source response is larger than 10 MiB", app.ErrInvalidInput)
	}
	mediaType := declaredType
	if mediaType == "" {
		mediaType = http.DetectContentType(content)
	}
	kind := mediaKindForType(mediaType)
	if kind == app.MediaKindAudio || kind == app.MediaKindVideo {
		return fetchedMediaSource{
			MediaType:         mediaType,
			MediaKind:         kind,
			ExternalVersion:   responseExternalVersion(resp.Header),
			ExternalUpdatedAt: responseLastModified(resp.Header),
			ByteSize:          normalizedContentLength(firstPositive(resp.ContentLength, int64(len(content)))),
		}, nil
	}
	if kind != app.MediaKindImage {
		return fetchedMediaSource{}, fmt.Errorf("%w: media URL content type %q is not supported", app.ErrInvalidInput, mediaType)
	}
	if !isPinnedImageMediaType(mediaType) {
		return fetchedMediaSource{}, fmt.Errorf("%w: image media type %q is not supported for pinning", app.ErrInvalidInput, mediaType)
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		return fetchedMediaSource{}, fmt.Errorf("%w: image source could not be decoded", app.ErrInvalidInput)
	}
	return fetchedMediaSource{
		Content:           content,
		MediaType:         mediaType,
		MediaKind:         app.MediaKindImage,
		ExternalVersion:   responseExternalVersion(resp.Header),
		ExternalUpdatedAt: responseLastModified(resp.Header),
		ByteSize:          int64(len(content)),
		Width:             config.Width,
		Height:            config.Height,
	}, nil
}

func fetchPDFSourceWithClient(ctx context.Context, rawURL string, client *http.Client) (fetchedPDFSource, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fetchedPDFSource{}, fmt.Errorf("%w: invalid PDF source URL", app.ErrInvalidInput)
	}
	req.Header.Set("Accept", "application/pdf,*/*;q=0.1")
	req.Header.Set("Accept-Language", "ko,en-US;q=0.9,en;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; PlasmaPDFFetcher/0.1; +https://github.com/c86j224s/liquid2)")
	resp, err := client.Do(req)
	if err != nil {
		return fetchedPDFSource{}, fmt.Errorf("%w: %s", app.ErrInvalidInput, sourceURLFetchFailureMessage(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fetchedPDFSource{}, fmt.Errorf("%w: %s", app.ErrInvalidInput, sourceURLHTTPStatusMessage(resp.StatusCode))
	}
	if resp.ContentLength > maxPDFSourceBytes {
		return fetchedPDFSource{}, fmt.Errorf("%w: PDF source response is larger than 100 MiB", app.ErrInvalidInput)
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, maxPDFSourceBytes+1))
	if err != nil {
		return fetchedPDFSource{}, err
	}
	if len(content) == 0 {
		return fetchedPDFSource{}, fmt.Errorf("%w: PDF source returned empty content", app.ErrInvalidInput)
	}
	if len(content) > maxPDFSourceBytes {
		return fetchedPDFSource{}, fmt.Errorf("%w: PDF source response is larger than 100 MiB", app.ErrInvalidInput)
	}
	mediaType := responseMediaType(resp.Header.Get("Content-Type"), content)
	if !pdftext.IsPDFMediaType(mediaType) && !pdftext.IsPDFBytes(content) {
		return fetchedPDFSource{}, fmt.Errorf("%w: PDF source content type %q is not supported", app.ErrInvalidInput, mediaType)
	}
	info, err := pdftext.Inspect(content)
	if err != nil {
		return fetchedPDFSource{}, fmt.Errorf("%w: PDF inspection failed: %v", app.ErrInvalidInput, err)
	}
	return fetchedPDFSource{
		Content:           content,
		MediaType:         pdftext.MediaType,
		Title:             pdfTitleFromURL(rawURL),
		ExternalVersion:   responseExternalVersion(resp.Header),
		ExternalUpdatedAt: responseLastModified(resp.Header),
		ByteSize:          int64(len(content)),
		PageCount:         info.PageCount,
		TextLengthKnown:   false,
	}, nil
}

func sourceURLFetchFailureMessage(err error) string {
	if errors.Is(err, context.Canceled) {
		return "URL 원문 가져오기가 취소되었습니다."
	}
	if errors.Is(err, context.DeadlineExceeded) || isTimeoutError(err) {
		return "URL 원문 응답이 제한 시간 내 도착하지 않았습니다. 원본 서버가 느리거나 자동 요청을 지연했을 수 있습니다. 잠시 뒤 다시 시도하거나 접근 가능한 다른 URL 또는 본문 텍스트를 소스로 추가하세요."
	}
	if strings.Contains(err.Error(), "blocked address") {
		return "URL 원문을 가져올 수 없습니다. source URL resolves to blocked address."
	}
	return "URL 원문을 가져올 수 없습니다. 네트워크 요청이 실패했습니다."
}

func isTimeoutError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func sourceURLHTTPStatusMessage(statusCode int) string {
	switch statusCode {
	case http.StatusUnauthorized:
		return "URL 원문을 가져올 수 없습니다. 원본 서버가 로그인이나 인증을 요구했습니다(HTTP 401). 이 후보는 자동으로 소스 스냅샷을 만들 수 없으니, 브라우저에서 열람 가능한 본문을 복사해 텍스트 소스로 추가하세요."
	case http.StatusForbidden:
		return "URL 원문을 가져올 수 없습니다. 원본 서버가 접근을 거부했습니다(HTTP 403). 이 후보는 자동으로 소스 스냅샷을 만들 수 없으니, 접근 가능한 다른 URL을 쓰거나 본문을 텍스트 소스로 추가하세요."
	case http.StatusNotFound:
		return "URL 원문을 가져올 수 없습니다. 원본 서버가 문서를 찾지 못했습니다(HTTP 404). URL이 맞는지 확인하거나 다른 소스 후보를 사용하세요."
	case http.StatusTooManyRequests:
		return "URL 원문을 가져올 수 없습니다. 원본 서버가 요청을 제한했습니다(HTTP 429). 잠시 뒤 다시 시도하거나 다른 소스 후보를 사용하세요."
	default:
		return fmt.Sprintf("URL 원문을 가져올 수 없습니다. 원본 서버가 HTTP %d 응답을 반환했습니다. 이 후보는 자동으로 소스 스냅샷을 만들 수 없습니다.", statusCode)
	}
}

func secureURLSourceHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: urlSourceDialTimeout}
	resolver := net.DefaultResolver
	return &http.Client{
		Timeout: urlSourceFetchTimeout,
		Transport: &http.Transport{
			Proxy:                  nil,
			DialContext:            secureURLSourceDialContext(dialer, resolver),
			TLSHandshakeTimeout:    urlSourceTLSHandshakeTimeout,
			ResponseHeaderTimeout:  urlSourceResponseHeaderTimeout,
			MaxResponseHeaderBytes: 64 << 10,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("stopped after 5 redirects")
			}
			if req.URL == nil || (req.URL.Scheme != "http" && req.URL.Scheme != "https") {
				return fmt.Errorf("redirected to a non-http URL")
			}
			if req.URL.User != nil {
				return fmt.Errorf("redirected to a URL with credentials")
			}
			return nil
		},
	}
}

func secureURLSourceDialContext(dialer *net.Dialer, resolver *net.Resolver) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid source URL address", app.ErrInvalidInput)
		}
		ips, err := resolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("%w: source URL host lookup failed: %v", app.ErrInvalidInput, err)
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("%w: source URL host has no addresses", app.ErrInvalidInput)
		}
		for _, ip := range ips {
			if isBlockedURLFetchIP(ip) {
				return nil, fmt.Errorf("%w: source URL resolves to blocked address %s", app.ErrInvalidInput, ip.String())
			}
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}
}

func isBlockedURLFetchIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() {
		return true
	}
	if ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() {
		return true
	}
	if cgnatPrefix().Contains(ip) {
		return true
	}
	return false
}

func cgnatPrefix() netip.Prefix {
	return netip.MustParsePrefix("100.64.0.0/10")
}

func responseMediaType(header string, content []byte) string {
	mediaType, params, err := mime.ParseMediaType(header)
	if err != nil || strings.TrimSpace(mediaType) == "" {
		return http.DetectContentType(content)
	}
	if charset := strings.TrimSpace(params["charset"]); charset != "" {
		return strings.ToLower(mediaType) + "; charset=" + charset
	}
	return strings.ToLower(mediaType)
}

func responseHeaderMediaType(header string) string {
	mediaType, params, err := mime.ParseMediaType(header)
	if err != nil || strings.TrimSpace(mediaType) == "" {
		return ""
	}
	if charset := strings.TrimSpace(params["charset"]); charset != "" {
		return strings.ToLower(mediaType) + "; charset=" + charset
	}
	return strings.ToLower(mediaType)
}

func responseLastModified(header http.Header) time.Time {
	if lastModified := strings.TrimSpace(header.Get("Last-Modified")); lastModified != "" {
		if parsed, err := http.ParseTime(lastModified); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func normalizedContentLength(length int64) int64 {
	if length < 0 {
		return 0
	}
	return length
}

func firstPositive(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func isTextualMediaType(mediaType string) bool {
	base, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		base = mediaType
	}
	base = strings.ToLower(strings.TrimSpace(base))
	return strings.HasPrefix(base, "text/") ||
		base == "application/json" ||
		base == "application/ld+json" ||
		base == "application/xml" ||
		base == "application/xhtml+xml" ||
		base == "application/rss+xml" ||
		base == "application/atom+xml" ||
		strings.HasSuffix(base, "+json") ||
		strings.HasSuffix(base, "+xml")
}

func mediaKindForType(mediaType string) string {
	base, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		base = mediaType
	}
	base = strings.ToLower(strings.TrimSpace(base))
	switch {
	case strings.HasPrefix(base, "image/"):
		return app.MediaKindImage
	case strings.HasPrefix(base, "audio/"):
		return app.MediaKindAudio
	case strings.HasPrefix(base, "video/"):
		return app.MediaKindVideo
	default:
		return ""
	}
}

func isPinnedImageMediaType(mediaType string) bool {
	base, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		base = mediaType
	}
	switch strings.ToLower(strings.TrimSpace(base)) {
	case "image/png", "image/jpeg", "image/gif":
		return true
	default:
		return false
	}
}

func mediaSourceReadNote(mediaKind string) string {
	switch strings.TrimSpace(mediaKind) {
	case app.MediaKindImage:
		return "이미지 원본은 source artifact로 저장되어 있지만, 현재 빌드에서는 이미지 내용 분석을 제공하지 않습니다. MCP와 웹 읽기는 메타데이터만 반환합니다."
	case app.MediaKindAudio, app.MediaKindVideo:
		return "오디오·영상은 현재 metadata/live-reference source로만 저장됩니다. inspect, 전사, 키프레임 추출은 아직 지원하지 않습니다."
	default:
		return "미디어 source 메타데이터만 반환합니다."
	}
}

func pdfTitleFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed == nil {
		return ""
	}
	name := path.Base(parsed.Path)
	if name == "." || name == "/" {
		return ""
	}
	name = strings.TrimSuffix(name, path.Ext(name))
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	return strings.TrimSpace(name)
}

func mediaLocatorFromJSON(raw json.RawMessage) (app.MediaLocator, error) {
	if len(raw) == 0 {
		return app.MediaLocator{}, fmt.Errorf("%w: media locator is required", app.ErrInvalidInput)
	}
	var locator app.MediaLocator
	if err := json.Unmarshal(raw, &locator); err == nil && locatorType(locator.LocatorType, locator.Kind) != "" {
		normalized, normalizeErr := normalizeWebMediaLocator(locator)
		if normalizeErr == nil {
			return normalized, nil
		}
		if locator, ok := uploadedImageMediaLocatorFromJSON(raw); ok {
			return locator, nil
		}
		return app.MediaLocator{}, normalizeErr
	}
	var locators []app.MediaLocator
	if err := json.Unmarshal(raw, &locators); err != nil {
		return app.MediaLocator{}, fmt.Errorf("%w: media locator must be an object or array", app.ErrInvalidInput)
	}
	for _, locator := range locators {
		if locatorType(locator.LocatorType, locator.Kind) == app.SourceLocatorTypeMedia {
			return normalizeWebMediaLocator(locator)
		}
	}
	if locator, ok := uploadedImageMediaLocatorFromJSON(raw); ok {
		return locator, nil
	}
	return app.MediaLocator{}, fmt.Errorf("%w: media locator is missing", app.ErrInvalidInput)
}

func uploadedImageMediaLocatorFromJSON(raw json.RawMessage) (app.MediaLocator, bool) {
	var locators []app.UploadedFileLocator
	if err := json.Unmarshal(raw, &locators); err != nil {
		var locator app.UploadedFileLocator
		if err := json.Unmarshal(raw, &locator); err != nil {
			return app.MediaLocator{}, false
		}
		locators = []app.UploadedFileLocator{locator}
	}
	for _, locator := range locators {
		discriminator := locatorType(locator.LocatorType, locator.Kind)
		mediaType := firstNonEmpty(locator.MIMEType, locator.MediaType)
		if discriminator != app.SourceConnectorTypeFileUpload {
			continue
		}
		if locator.ContentKind != app.UploadedContentKindImage && !strings.HasPrefix(mediaType, "image/") {
			continue
		}
		return app.MediaLocator{
			LocatorType: app.SourceLocatorTypeMedia,
			MediaKind:   app.MediaKindImage,
			Provider:    app.SourceConnectorTypeFileUpload,
			MIMEType:    mediaType,
			ByteSize:    locator.ByteSize,
			Title:       firstNonEmpty(locator.SanitizedFilename, locator.OriginalFilename),
			SHA256:      locator.SHA256,
		}, true
	}
	return app.MediaLocator{}, false
}

func normalizeWebMediaLocator(locator app.MediaLocator) (app.MediaLocator, error) {
	discriminator := locatorType(locator.LocatorType, locator.Kind)
	if discriminator != app.SourceLocatorTypeMedia {
		return app.MediaLocator{}, fmt.Errorf("%w: media locator kind is invalid", app.ErrInvalidInput)
	}
	locator.MediaKind = strings.TrimSpace(locator.MediaKind)
	switch locator.MediaKind {
	case app.MediaKindImage, app.MediaKindAudio, app.MediaKindVideo:
	default:
		return app.MediaLocator{}, fmt.Errorf("%w: media locator media_kind is invalid", app.ErrInvalidInput)
	}
	locator.LocatorType = app.SourceLocatorTypeMedia
	locator.Kind = ""
	return locator, nil
}

func locatorType(locatorType string, legacyKind string) string {
	if trimmed := strings.TrimSpace(locatorType); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(legacyKind)
}

func htmlTitle(content []byte, mediaType string) string {
	base, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		base = mediaType
	}
	if base != "text/html" && base != "application/xhtml+xml" {
		return ""
	}
	match := regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`).FindSubmatch(content)
	if len(match) < 2 {
		return ""
	}
	title := htmlpkg.UnescapeString(string(match[1]))
	title = strings.Join(strings.Fields(title), " ")
	if len(title) > 160 {
		title = title[:160]
	}
	return strings.TrimSpace(title)
}

func responseExternalVersion(header http.Header) string {
	values := []string{}
	if etag := strings.TrimSpace(header.Get("ETag")); etag != "" {
		values = append(values, "etag="+etag)
	}
	if modified := strings.TrimSpace(header.Get("Last-Modified")); modified != "" {
		values = append(values, "last-modified="+modified)
	}
	return strings.Join(values, "; ")
}

func sourceFileExtension(mediaType string) string {
	base, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		base = mediaType
	}
	switch strings.ToLower(strings.TrimSpace(base)) {
	case "text/html", "application/xhtml+xml":
		return ".html"
	case "application/pdf":
		return ".pdf"
	case "application/json", "application/ld+json":
		return ".json"
	case "application/xml", "application/rss+xml", "application/atom+xml":
		return ".xml"
	default:
		return ".txt"
	}
}

func sourceCandidatesFromText(text string) []sourceCandidate {
	parsed := sourcecandidates.Parse(text)
	if len(parsed) == 0 {
		return nil
	}
	candidates := make([]sourceCandidate, 0, len(parsed))
	for _, candidate := range parsed {
		candidates = append(candidates, sourceCandidate{
			URL:    candidate.URL,
			Title:  candidate.Title,
			Reason: candidate.Reason,
			State:  candidate.State,
		})
	}
	return candidates
}

func safeFilename(title string, ext string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "source"
	}
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ' || r == '.':
			b.WriteRune('-')
		}
	}
	name := strings.Trim(b.String(), "-_")
	if name == "" {
		name = "source"
	}
	if len(name) > 80 {
		name = name[:80]
	}
	return name + ext
}
