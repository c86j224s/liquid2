package web

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	promptUnsafeImageDataURIRegexp = regexp.MustCompile(`(?is)data:image/[a-z0-9.+-]+(?:\s*;\s*[^,\s;]+(?:=[^,\s;]+)?)*\s*;\s*base64\s*,\s*(?:[a-z0-9+/=_-]\s*)+`)
	promptUnsafeLongBase64Regexp   = regexp.MustCompile(`\b[A-Za-z0-9+/]{256,}={0,2}\b`)
)

const (
	designedReportRendererVersion = "dh30-source-tex-brackets-20260713"
)

type reportDesignRequest struct {
	AgentExecutor        string `json:"agent_executor"`
	AgentModel           string
	AgentReasoningEffort string
}

type designedReportContentModel struct {
	Kicker           string                         `json:"kicker"`
	Title            string                         `json:"title"`
	Subtitle         string                         `json:"subtitle"`
	Thesis           string                         `json:"thesis"`
	Markers          []designedReportMarker         `json:"markers"`
	HeroVisual       designedReportHero             `json:"hero_visual"`
	VisualUnits      []designedReportVisual         `json:"visual_units"`
	Tabs             []designedReportTab            `json:"tabs"`
	Sources          []designedReportSource         `json:"sources"`
	Caveats          []string                       `json:"caveats"`
	Glossary         []designedReportGlossary       `json:"glossary"`
	VisualIdentity   designedReportVisualIdentity   `json:"visual_identity"`
	CompositionShape designedReportCompositionShape `json:"composition_shape"`
}

type designedReportMarker struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Note  string `json:"note"`
}

type designedReportHero struct {
	Title      string               `json:"title"`
	LeftLabel  string               `json:"left_label"`
	RightLabel string               `json:"right_label"`
	Nodes      []designedReportNode `json:"nodes"`
}

type designedReportVisual struct {
	Title    string               `json:"title"`
	Kind     string               `json:"kind"`
	Question string               `json:"question"`
	Nodes    []designedReportNode `json:"nodes"`
	Caption  string               `json:"caption"`
}

type designedReportNode struct {
	Label string `json:"label"`
	Body  string `json:"body"`
	Tone  string `json:"tone"`
}

type designedReportTab struct {
	Label    string                  `json:"label"`
	Question string                  `json:"question"`
	Summary  string                  `json:"summary"`
	Takeaway string                  `json:"takeaway"`
	Sections []designedReportSection `json:"sections"`
}

type designedReportSection struct {
	Heading    string                         `json:"heading"`
	Body       []string                       `json:"body"`
	Bullets    []string                       `json:"bullets"`
	Component  string                         `json:"component"`
	Table      designedReportTable            `json:"table"`
	Diagram    designedReportDiagram          `json:"diagram"`
	Images     []designedReportImagePlacement `json:"images"`
	Caveat     string                         `json:"caveat"`
	SourceNote string                         `json:"source_note"`
}

type designedReportImagePlacement struct {
	ImageRef  string `json:"image_ref"`
	Caption   string `json:"caption"`
	Placement string `json:"placement"`
}

type designedReportTable struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

type designedReportDiagram struct {
	Title string               `json:"title"`
	Steps []designedReportNode `json:"steps"`
}

type designedReportSource struct {
	Label string `json:"label"`
	Href  string `json:"href"`
	Note  string `json:"note"`
}

type designedReportGlossary struct {
	Term       string `json:"term"`
	Definition string `json:"definition"`
}

type designedReportVisualIdentity struct {
	StyleKey        string `json:"style_key"`
	Motif           string `json:"motif"`
	PaletteNote     string `json:"palette_note"`
	InteractionNote string `json:"interaction_note"`
}

type designedReportCompositionShape struct {
	ShapeKey            string `json:"shape_key"`
	Rationale           string `json:"rationale"`
	PrimaryReaderAction string `json:"primary_reader_action"`
}

func (image reportInlineImage) Caption() string {
	parts := []string{}
	if image.Width > 0 && image.Height > 0 {
		parts = append(parts, fmt.Sprintf("%dx%d", image.Width, image.Height))
	}
	if image.License != "" {
		parts = append(parts, "license: "+image.License)
	}
	if image.Attribution != "" {
		parts = append(parts, "attribution: "+image.Attribution)
	}
	if image.SourceURL != "" {
		parts = append(parts, image.SourceURL)
	}
	if len(parts) == 0 {
		parts = append(parts, image.SnapshotID)
	}
	return strings.Join(parts, " / ")
}
