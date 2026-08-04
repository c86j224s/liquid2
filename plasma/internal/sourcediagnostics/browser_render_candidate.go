package sourcediagnostics

import (
	"mime"
	"strings"
)

const (
	browserRenderCandidateMaxVisibleText = 800
	browserRenderCandidateMinHTMLBytes   = 1500
)

// BrowserRenderCandidateDiagnosis는 일반 URL fetch 결과가 브라우저 렌더링
// 재시도 후보인지 나타내는 진단값이다. Candidate가 false이면 다른 필드는
// 호출자가 표시하지 않아도 되는 설명 보조 정보다.
type BrowserRenderCandidateDiagnosis struct {
	Candidate         bool     `json:"candidate"`
	Reason            string   `json:"reason"`
	VisibleTextLength int      `json:"visible_text_length"`
	HTMLByteSize      int      `json:"html_byte_size"`
	Signals           []string `json:"signals,omitempty"`
}

// DiagnoseBrowserRenderCandidate는 이미 가져온 HTML 본문만 검사해 브라우저
// 렌더링 후보 여부를 판정한다. 네트워크 호출이나 브라우저 실행은 하지 않으며,
// 봇 차단 화면이나 충분히 읽을 수 있는 정적 HTML은 후보로 표시하지 않는다.
func DiagnoseBrowserRenderCandidate(content []byte, mediaType string) BrowserRenderCandidateDiagnosis {
	if !browserRenderCandidateHTMLMediaType(mediaType) || len(content) == 0 {
		return BrowserRenderCandidateDiagnosis{}
	}
	stats := inspectBrowserRenderHTML(content)
	if stats.HasBotWall || stats.VisibleTextLength > browserRenderCandidateMaxVisibleText || len(content) < browserRenderCandidateMinHTMLBytes {
		return BrowserRenderCandidateDiagnosis{}
	}
	signals := browserRenderCandidateSignals(stats, len(content))
	if len(signals) < 2 {
		return BrowserRenderCandidateDiagnosis{}
	}
	return BrowserRenderCandidateDiagnosis{
		Candidate:         true,
		Reason:            "현재 URL fetch 본문이 JavaScript app shell에 가까워 브라우저 렌더링 검증 후보로 표시합니다.",
		VisibleTextLength: stats.VisibleTextLength,
		HTMLByteSize:      len(content),
		Signals:           signals,
	}
}

func browserRenderCandidateHTMLMediaType(mediaType string) bool {
	base, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		base = mediaType
	}
	switch strings.ToLower(strings.TrimSpace(base)) {
	case "text/html", "application/xhtml+xml":
		return true
	default:
		return false
	}
}

func browserRenderCandidateSignals(stats browserRenderHTMLStats, byteSize int) []string {
	signals := []string{"short_visible_text"}
	if stats.ScriptCount >= 3 || stats.ExternalScripts >= 2 {
		signals = append(signals, "script_heavy_html")
	}
	if stats.HasAppMount {
		signals = append(signals, "app_mount_node")
	}
	if stats.HasHydration {
		signals = append(signals, "hydration_marker")
	}
	if stats.HasJSRequiredText {
		signals = append(signals, "javascript_required_text")
	}
	if byteSize > 0 && stats.VisibleTextLength*8 < byteSize {
		signals = append(signals, "thin_visible_text_vs_html")
	}
	return signals
}
