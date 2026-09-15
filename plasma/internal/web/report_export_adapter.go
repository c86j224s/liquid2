package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting/reportdocument"
	htmlpkg "html"
	"strconv"
	"strings"
	"time"
)

func (server *Server) exportMarkdownArtifactAsHTML(ctx context.Context, missionID string, sourceArtifact artifactcontract.Raw) (reportdocument.ReportExportResult, error) {
	unlockReports := server.reports.lock(missionID)
	defer unlockReports()
	refetched, err := server.reportArtifact(ctx, missionID, sourceArtifact.ArtifactID)
	if err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	sourceArtifact = refetched
	if !isMarkdownMediaType(sourceArtifact.MediaType) {
		return reportdocument.ReportExportResult{}, fmt.Errorf("%w: HTML export requires a Markdown report artifact", producterror.ErrInvalidInput)
	}
	if cached, ok, err := server.existingMarkdownArtifactHTMLExport(ctx, missionID, sourceArtifact.ArtifactID); err != nil {
		return reportdocument.ReportExportResult{}, err
	} else if ok {
		return cached, nil
	}
	article, err := server.isArticleArtifact(ctx, missionID, sourceArtifact.ArtifactID)
	if err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	content, err := server.renderSelfContainedReportHTML(ctx, missionID, sourceArtifact, article)
	if err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	artifact, event, err := server.service.CreateRawArtifactWithEvent(ctx, artifactcontract.CreateRequest{
		ArtifactID: newID("art"),
		MissionID:  missionID,
		MediaType:  "text/html; charset=utf-8",
		Filename:   markdownReportHTMLFilename(sourceArtifact),
		Producer:   ledger.Producer{Type: "plasma", ID: "html-export"},
		Content:    content,
	}, func(artifact artifactcontract.Raw) ledger.AppendRequest {
		return reportexecution.BuildSelfContainedHTMLExportAppendRequest(reportexecution.SelfContainedHTMLExportEventRequest{
			EventID:          newID("evt"),
			MissionID:        missionID,
			SourceArtifactID: sourceArtifact.ArtifactID,
			Artifact:         artifact,
			RendererVersion:  selfContainedReportRendererVersion,
			Producer:         ledger.Producer{Type: "plasma", ID: "html-export"},
		})
	})
	if err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	return reportdocument.ReportExportResult{Artifact: artifact, Event: event}, nil
}

func (server *Server) existingMarkdownArtifactHTMLExport(ctx context.Context, missionID string, sourceArtifactID string) (reportdocument.ReportExportResult, bool, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return reportdocument.ReportExportResult{}, false, err
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.EventType != "report.artifact.exported" {
			continue
		}
		var payload struct {
			Kind             string `json:"kind"`
			SourceArtifactID string `json:"source_artifact_id"`
			ArtifactID       string `json:"artifact_id"`
			Target           string `json:"target"`
			RendererVersion  string `json:"renderer_version"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.Kind) != reportexecution.ExportKindSelfContainedHTML ||
			strings.TrimSpace(payload.SourceArtifactID) != sourceArtifactID ||
			strings.TrimSpace(payload.Target) != reportexecution.ExportTargetSelfContainedHTML ||
			strings.TrimSpace(payload.RendererVersion) != selfContainedReportRendererVersion {
			continue
		}
		artifactID := strings.TrimSpace(payload.ArtifactID)
		if artifactID == "" {
			continue
		}
		artifact, err := server.service.GetRawArtifact(ctx, artifactID)
		if err != nil {
			return reportdocument.ReportExportResult{}, false, err
		}
		return reportdocument.ReportExportResult{Artifact: artifact, Event: event}, true, nil
	}
	return reportdocument.ReportExportResult{}, false, nil
}

func (server *Server) createDesignedReportHTMLExport(ctx context.Context, missionID string, sourceArtifactID string, req reportDesignRequest, pendingEventID string) (reportdocument.ReportExportResult, error) {
	sourceArtifact, err := server.reportArtifact(ctx, missionID, sourceArtifactID)
	if err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	if !isMarkdownMediaType(sourceArtifact.MediaType) {
		return reportdocument.ReportExportResult{}, fmt.Errorf("%w: designed HTML export requires a Markdown report artifact", producterror.ErrInvalidInput)
	}
	images, notes, err := server.inlineReportImages(ctx, missionID)
	if err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	imageSetFingerprint := designedReportImageSetFingerprint(images, notes)
	if cached, ok, err := server.existingDesignedReportHTMLExport(ctx, missionID, sourceArtifact.ArtifactID, imageSetFingerprint); err != nil {
		return reportdocument.ReportExportResult{}, err
	} else if ok {
		return cached, nil
	}
	executorName, err := normalizeAgentExecutorName(req.AgentExecutor)
	if err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	executor := server.agentExecutor(executorName)
	if executor == nil {
		return reportdocument.ReportExportResult{}, fmt.Errorf("%w: designed HTML export requires an agent executor", producterror.ErrInvalidInput)
	}
	agentModel := strings.TrimSpace(req.AgentModel)
	agentReasoningEffort := strings.TrimSpace(req.AgentReasoningEffort)
	agentModel, agentReasoningEffort, err = resolveAgentSettings(executorName, agentModel, agentReasoningEffort, "")
	if err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	title := reportArtifactTitle(sourceArtifact)
	toolSessionID := newID("ses")
	started := time.Now()
	result, err := executor.Run(ctx, AgentRequest{
		UserText:        "generate designed HTML content model",
		Prompt:          agentDesignedHTMLContentModelPrompt(title, string(sourceArtifact.Content), images),
		Model:           agentModel,
		ReasoningEffort: agentReasoningEffort,
		MissionID:       missionID,
		ToolSessionID:   toolSessionID,
		AgentExecutor:   executorName,
		MCPMode:         "auto",
	})
	agentDurationMS := time.Since(started).Milliseconds()
	if err != nil {
		return reportdocument.ReportExportResult{}, fmt.Errorf("designed HTML content model agent failed: %w", reportAgentFailure(err, result, "report_design", agentDurationMS, ""))
	}
	model, modelJSON, err := parseDesignedReportContentModel(result.Text)
	if err != nil {
		return reportdocument.ReportExportResult{}, reportAgentFailure(err, result, "report_design", agentDurationMS, "")
	}
	content, err := server.renderDesignedReportHTML(sourceArtifact, model, images, notes)
	if err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	producer := ledger.Producer{Type: "agent_session", ID: fallbackSessionID(result.SessionID, toolSessionID)}
	_, artifact, event, closed, err := server.service.CreateDesignedReportHTMLExportIfOpen(ctx, missionID, pendingEventID, artifactcontract.CreateRequest{
		ArtifactID: newID("art"),
		MissionID:  missionID,
		MediaType:  "application/json; charset=utf-8",
		Filename:   safeFilename(title+" content model", ".json"),
		Producer:   producer,
		Content:    modelJSON,
	}, artifactcontract.CreateRequest{
		ArtifactID: newID("art"),
		MissionID:  missionID,
		MediaType:  "text/html; charset=utf-8",
		Filename:   safeFilename(title+" designed", ".html"),
		Producer:   producer,
		Content:    content,
	}, func(modelArtifact artifactcontract.Raw, artifact artifactcontract.Raw) ledger.AppendRequest {
		return reportexecution.BuildDesignedHTMLExportAppendRequest(reportexecution.DesignedHTMLExportEventRequest{
			EventID:                newID("evt"),
			MissionID:              missionID,
			PendingEventID:         pendingEventID,
			SourceArtifactID:       sourceArtifact.ArtifactID,
			ContentModelArtifactID: modelArtifact.ArtifactID,
			Artifact:               artifact,
			RendererVersion:        designedReportRendererVersion,
			ImageSetFingerprint:    imageSetFingerprint,
			AgentExecutor:          executorName,
			AgentModel:             agentModel,
			AgentReasoningEffort:   agentReasoningEffort,
			AgentSessionID:         result.SessionID,
			ToolSessionID:          toolSessionID,
			DurationMS:             time.Since(started).Milliseconds(),
			AgentDurationMS:        agentDurationMS,
			AgentUsage:             result.Usage,
			AgentResumed:           result.Resumed,
			Producer:               producer,
		})
	})
	if err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	if !closed {
		return reportdocument.ReportExportResult{}, fmt.Errorf("%w: designed HTML report operation is already closed", producterror.ErrConflict)
	}
	return reportdocument.ReportExportResult{Artifact: artifact, Event: event}, nil
}

func (server *Server) reconcileStaleDesignedReportExports(ctx context.Context, missionID string) error {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return err
	}
	completed := reportexecution.CompletedPendingEventIDs(events)
	for _, event := range events {
		if event.EventType != "report.design.pending" {
			continue
		}
		if _, ok := completed[event.EventID]; ok || server.runningReports.Owns(missionID, event.EventID) {
			continue
		}
		return server.reportRunner().ResumeDesign(ctx, missionID, event)
	}
	return nil
}

func (server *Server) existingDesignedReportHTMLExport(ctx context.Context, missionID string, sourceArtifactID string, imageSetFingerprint string) (reportdocument.ReportExportResult, bool, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return reportdocument.ReportExportResult{}, false, err
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.EventType != "report.artifact.exported" {
			continue
		}
		var payload struct {
			Kind             string `json:"kind"`
			SourceArtifactID string `json:"source_artifact_id"`
			ArtifactID       string `json:"artifact_id"`
			Target           string `json:"target"`
			RendererVersion  string `json:"renderer_version"`
			ImageSet         string `json:"image_set_fingerprint"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.Kind) != reportexecution.ExportKindDesignedHTML ||
			strings.TrimSpace(payload.SourceArtifactID) != sourceArtifactID ||
			strings.TrimSpace(payload.Target) != reportexecution.ExportTargetDesignedHTML ||
			strings.TrimSpace(payload.RendererVersion) != designedReportRendererVersion ||
			strings.TrimSpace(payload.ImageSet) != imageSetFingerprint {
			continue
		}
		artifactID := strings.TrimSpace(payload.ArtifactID)
		if artifactID == "" {
			continue
		}
		artifact, err := server.service.GetRawArtifact(ctx, artifactID)
		if err != nil {
			return reportdocument.ReportExportResult{}, false, err
		}
		return reportdocument.ReportExportResult{Artifact: artifact, Event: event}, true, nil
	}
	return reportdocument.ReportExportResult{}, false, nil
}

func (server *Server) renderSelfContainedReportHTML(ctx context.Context, missionID string, sourceArtifact artifactcontract.Raw, article bool) ([]byte, error) {
	images, notes, err := server.inlineReportImages(ctx, missionID)
	if err != nil {
		return nil, err
	}
	title := reportArtifactTitle(sourceArtifact)
	mathHead, err := selfContainedMathHead()
	if err != nil {
		return nil, err
	}
	mermaidHead, err := selfContainedMermaidHead()
	if err != nil {
		return nil, err
	}
	mathScripts, err := selfContainedMarkdownScripts()
	if err != nil {
		return nil, err
	}
	wordCount := len(strings.Fields(string(sourceArtifact.Content)))
	markdownJSON, err := json.Marshal(string(sourceArtifact.Content))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.WriteString("<!doctype html>\n<html lang=\"ko\">\n<head>\n<meta charset=\"utf-8\">\n")
	out.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	out.WriteString("<title>" + htmlpkg.EscapeString(title) + "</title>\n")
	out.WriteString(selfContainedReportCSS())
	if article {
		out.WriteString("<style>.article-output .layout{display:block;max-width:52rem}.article-output .report-body{margin:0 auto}.article-output .report-body>:is(h1,h2,h3,h4,h5,h6,p,ul,ol,blockquote){max-width:68ch}.article-output .media-panel,.article-output .notes{max-width:68ch;margin-left:auto;margin-right:auto}</style>")
	}
	out.WriteString(mathHead)
	out.WriteString(mermaidHead)
	out.WriteString(selfContainedBasicMermaidCSS())
	bodyClass, eyebrow, subtitle := "", "Plasma Report", "Markdown report artifact에서 파생한 self-contained interactive HTML입니다. 이미지는 가능한 경우 원본 source artifact를 data URI로 포함했습니다."
	if article {
		bodyClass, eyebrow, subtitle = " class=\"article-output\"", "Plasma Article", "승인된 소스를 바탕으로 만든 독자용 글입니다."
	}
	out.WriteString("</head>\n<body" + bodyClass + ">\n")
	out.WriteString("<header class=\"hero\"><div><p class=\"eyebrow\">" + eyebrow + "</p><h1>" + htmlpkg.EscapeString(title) + "</h1><p class=\"sub\">" + subtitle + "</p></div><button id=\"themeToggle\" type=\"button\" hidden aria-pressed=\"false\" aria-label=\"다크 모드 켜기\">다크 모드</button></header>\n")
	out.WriteString("<main class=\"layout\">\n")
	if !article {
		out.WriteString("<aside class=\"rail\"><div class=\"metric\"><span>본문 단어</span><strong>" + strconv.Itoa(wordCount) + "</strong></div><div class=\"metric\"><span>포함 이미지</span><strong>" + strconv.Itoa(len(images)) + "</strong></div><div class=\"metric\"><span>원본 artifact</span><code>" + htmlpkg.EscapeString(sourceArtifact.ArtifactID) + "</code></div><nav><a href=\"#report-body\">본문</a><a href=\"#media-gallery\">미디어</a><a href=\"#export-notes\">생성 노트</a></nav></aside>\n")
	} else {
		_ = wordCount
	}
	out.WriteString("<article id=\"report-body\" class=\"report-body\">\n")
	out.WriteString("<pre class=\"report-markdown-raw\">" + htmlpkg.EscapeString(string(sourceArtifact.Content)) + "</pre>")
	out.WriteString("</article>\n")
	out.WriteString("<script id=\"report-markdown\" type=\"application/json\">" + string(markdownJSON) + "</script>\n")
	out.WriteString("<section id=\"media-gallery\" class=\"media-panel\"><div class=\"section-head\"><h2>미디어</h2><span>" + strconv.Itoa(len(images)) + "개 이미지 포함</span></div>")
	if len(images) == 0 {
		out.WriteString("<p class=\"muted\">이 미션의 active image source 중 self-contained HTML에 포함할 수 있는 이미지가 없습니다.</p>")
	} else {
		out.WriteString("<div class=\"gallery\">")
		for _, image := range images {
			out.WriteString("<figure><img loading=\"lazy\" src=\"" + image.DataURI + "\" alt=\"" + htmlpkg.EscapeString(image.Title) + "\"><figcaption><strong>" + htmlpkg.EscapeString(image.Title) + "</strong><span>" + htmlpkg.EscapeString(image.Caption()) + "</span></figcaption></figure>")
		}
		out.WriteString("</div>")
	}
	out.WriteString("</section>\n")
	out.WriteString("<section id=\"export-notes\" class=\"notes\"><h2>생성 노트</h2><ul>")
	if article {
		out.WriteString("<li>이 HTML은 글 내용을 다시 생성하지 않고 저장된 Markdown artifact를 렌더링했습니다.</li>")
	} else {
		out.WriteString("<li>이 HTML은 보고서 내용을 다시 생성하지 않고 저장된 Markdown artifact를 렌더링했습니다.</li>")
	}
	out.WriteString("<li>오디오와 영상은 self-contained로 포함하지 않습니다.</li>")
	for _, note := range notes {
		out.WriteString("<li>" + htmlpkg.EscapeString(note) + "</li>")
	}
	out.WriteString("</ul></section>\n")
	out.WriteString("</main>\n")
	out.WriteString(selfContainedReportThemeScript())
	out.WriteString(mathScripts)
	out.WriteString("</body>\n</html>\n")
	return out.Bytes(), nil
}
