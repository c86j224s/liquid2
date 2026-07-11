package urlsource

import (
	"context"
	"errors"
	"fmt"
	htmlpkg "html"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/sources/pdftext"
)

const (
	MaxTextSourceBytes            = 20 << 20
	MaxPDFSourceBytes             = 100 << 20
	sourceDialTimeout             = 15 * time.Second
	sourceTLSHandshakeTimeout     = 15 * time.Second
	sourceResponseHeaderTimeout   = 45 * time.Second
	sourceFetchTimeout            = 60 * time.Second
	sourceMaxResponseHeaderBytes  = 64 << 10
	sourceMaxRedirects            = 5
	sourceFetcherUserAgentVersion = "0.1"
)

type Fetched struct {
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

func Fetch(ctx context.Context, rawURL string) (Fetched, error) {
	return FetchWithClient(ctx, rawURL, secureHTTPClient())
}

func FetchWithClient(ctx context.Context, rawURL string, client *http.Client) (Fetched, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return Fetched{}, fmt.Errorf("%w: invalid source URL", app.ErrInvalidInput)
	}
	if req.URL.User != nil {
		return Fetched{}, fmt.Errorf("%w: source URL must not include credentials", app.ErrInvalidInput)
	}
	req.Header.Set("Accept", "application/pdf,text/html,text/plain,application/json,application/xml,text/*,*/*;q=0.1")
	req.Header.Set("Accept-Language", "ko,en-US;q=0.9,en;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; PlasmaSourceFetcher/"+sourceFetcherUserAgentVersion+"; +https://github.com/c86j224s/liquid2)")
	resp, err := client.Do(req)
	if err != nil {
		return Fetched{}, fmt.Errorf("%w: %s", app.ErrInvalidInput, fetchFailureMessage(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Fetched{}, fmt.Errorf("%w: %s", app.ErrInvalidInput, httpStatusMessage(resp.StatusCode))
	}
	declaredType := responseHeaderMediaType(resp.Header.Get("Content-Type"))
	pdfExpected := pdftext.IsPDFMediaType(declaredType) || strings.HasSuffix(strings.ToLower(resp.Request.URL.Path), ".pdf")
	limit := MaxTextSourceBytes
	limitLabel := "20 MiB"
	if pdfExpected {
		limit = MaxPDFSourceBytes
		limitLabel = "100 MiB"
	}
	if resp.ContentLength > int64(limit) {
		return Fetched{}, fmt.Errorf("%w: source URL response is larger than %s", app.ErrInvalidInput, limitLabel)
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, int64(limit)+1))
	if err != nil {
		return Fetched{}, err
	}
	if len(content) == 0 {
		return Fetched{}, fmt.Errorf("%w: source URL returned empty content", app.ErrInvalidInput)
	}
	if len(content) > limit {
		return Fetched{}, fmt.Errorf("%w: source URL response is larger than %s", app.ErrInvalidInput, limitLabel)
	}
	mediaType := responseMediaType(resp.Header.Get("Content-Type"), content)
	if pdftext.IsPDFMediaType(mediaType) || pdftext.IsPDFBytes(content) {
		if len(content) > MaxPDFSourceBytes {
			return Fetched{}, fmt.Errorf("%w: source URL response is larger than 100 MiB", app.ErrInvalidInput)
		}
		info, err := pdftext.Inspect(content)
		if err != nil {
			return Fetched{}, fmt.Errorf("%w: PDF inspection failed: %v", app.ErrInvalidInput, err)
		}
		return Fetched{
			Content:           content,
			MediaType:         pdftext.MediaType,
			Title:             titleFromURL(resp.Request.URL),
			ExternalVersion:   responseExternalVersion(resp.Header),
			ExternalUpdatedAt: responseLastModified(resp.Header),
			ByteSize:          int64(len(content)),
			PageCount:         info.PageCount,
			TextLengthKnown:   false,
		}, nil
	}
	if len(content) > MaxTextSourceBytes {
		return Fetched{}, fmt.Errorf("%w: source URL response is larger than 20 MiB", app.ErrInvalidInput)
	}
	if !isTextualMediaType(mediaType) {
		return Fetched{}, fmt.Errorf("%w: source URL content type %q is not supported", app.ErrInvalidInput, mediaType)
	}
	if looksLikeAtlassianAuthWall(resp.Request.URL, content, mediaType) {
		return Fetched{}, fmt.Errorf("%w: URL 원문이 Atlassian 로그인 화면으로 확인되었습니다. Confluence 자료는 Settings에서 API token 연결을 만든 뒤 Sources의 Confluence에서 page를 검색해 추가하세요.", app.ErrInvalidInput)
	}
	return Fetched{
		Content:           content,
		MediaType:         mediaType,
		Title:             htmlTitle(content, mediaType),
		ExternalVersion:   responseExternalVersion(resp.Header),
		ExternalUpdatedAt: responseLastModified(resp.Header),
		ByteSize:          int64(len(content)),
		TextLengthKnown:   true,
		TextLength:        len(content),
	}, nil
}

func SourceFileExtension(mediaType string) string {
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

func titleFromURL(parsed *url.URL) string {
	if parsed == nil {
		return ""
	}
	base := strings.TrimSpace(pathBase(parsed.Path))
	if base == "" || base == "." || base == "/" {
		return parsed.String()
	}
	return base
}

func pathBase(value string) string {
	value = strings.TrimRight(value, "/")
	if value == "" {
		return ""
	}
	idx := strings.LastIndex(value, "/")
	if idx >= 0 {
		return value[idx+1:]
	}
	return value
}

func fetchFailureMessage(err error) string {
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

func httpStatusMessage(statusCode int) string {
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

func secureHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: sourceDialTimeout}
	resolver := net.DefaultResolver
	return &http.Client{
		Timeout: sourceFetchTimeout,
		Transport: &http.Transport{
			Proxy:                  nil,
			DialContext:            secureDialContext(dialer, resolver),
			TLSHandshakeTimeout:    sourceTLSHandshakeTimeout,
			ResponseHeaderTimeout:  sourceResponseHeaderTimeout,
			MaxResponseHeaderBytes: sourceMaxResponseHeaderBytes,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= sourceMaxRedirects {
				return fmt.Errorf("stopped after %d redirects", sourceMaxRedirects)
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

func secureDialContext(dialer *net.Dialer, resolver *net.Resolver) func(context.Context, string, string) (net.Conn, error) {
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
			if isBlockedFetchIP(ip) {
				return nil, fmt.Errorf("%w: source URL resolves to blocked address %s", app.ErrInvalidInput, ip.String())
			}
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}
}

func isBlockedFetchIP(ip netip.Addr) bool {
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
	return netip.MustParsePrefix("100.64.0.0/10").Contains(ip)
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
