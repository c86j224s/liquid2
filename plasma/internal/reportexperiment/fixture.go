package reportexperiment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const FixtureSchemaVersion = "plasma.reportexperiment.fixed_parts.v1"

// Fixture는 archive에 저장하는 reviewed Part 입력 계약이다.
//
// Part 본문은 JSON 안에 두지 않고 file path와 SHA-256 receipt만 둔다. 실행기는
// seed 전에 모든 Part byte와 index 순서를 검증한다.
type Fixture struct {
	SchemaVersion             string                           `json:"schema_version"`
	FixtureID                 string                           `json:"fixture_id"`
	SourceProvenance          SourceProvenance                 `json:"source_provenance"`
	ReportTitle               string                           `json:"report_title"`
	Rigor                     FixtureRigor                     `json:"rigor"`
	DirectionHint             string                           `json:"direction_hint"`
	WritingContract           *reporting.ReportWritingContract `json:"writing_contract"`
	GenerationGuidanceProfile string                           `json:"generation_guidance_profile,omitempty"`
	GenerationGuidanceSHA256  string                           `json:"generation_guidance_sha256,omitempty"`
	PostReportHumanize        string                           `json:"post_report_humanize"`
	Parts                     []FixturePart                    `json:"parts"`
}

// SourceProvenance는 fixture가 어느 검토 입력에서 왔는지 식별하는 공개 receipt다.
type SourceProvenance struct {
	ProvenanceID  string `json:"provenance_id"`
	ProductCommit string `json:"product_commit"`
	SourceID      string `json:"source_id,omitempty"`
	Notes         string `json:"notes,omitempty"`
}

// FixtureRigor는 제품 prompt에 전달할 엄격도 표시값이다.
type FixtureRigor struct {
	Level string `json:"level"`
	Label string `json:"label"`
}

// FixturePart는 archive-local reviewed Part 파일 하나의 위치와 receipt다.
type FixturePart struct {
	Index        int    `json:"index"`
	Title        string `json:"title"`
	Path         string `json:"path"`
	SHA256       string `json:"sha256"`
	SectionTitle string `json:"section_title,omitempty"`
}

// LoadedFixture는 검증을 통과해 실행에 사용할 수 있는 fixture와 Part bytes다.
type LoadedFixture struct {
	Spec        Fixture
	FixturePath string
	SHA256      string
	Parts       []LoadedPart
}

// LoadedPart는 fixture Part receipt와 실제 bytes를 함께 보존한다.
type LoadedPart struct {
	Spec      FixturePart
	AbsPath   string
	SHA256    string
	Content   []byte
	WordCount int
}

func loadFixture(archiveRoot, fixturePath, repoRoot string) (LoadedFixture, error) {
	archiveRoot, err := canonicalExistingPath(archiveRoot)
	if err != nil {
		return LoadedFixture{}, err
	}
	fixtureAbs, err := canonicalExistingPath(fixturePath)
	if err != nil {
		return LoadedFixture{}, err
	}
	if !pathInside(archiveRoot, fixtureAbs) {
		return LoadedFixture{}, fmt.Errorf("%w: fixture must be under archive root", producterror.ErrInvalidInput)
	}
	if err := rejectInsideRepo(repoRoot, fixtureAbs); err != nil {
		return LoadedFixture{}, err
	}
	raw, err := os.ReadFile(fixtureAbs)
	if err != nil {
		return LoadedFixture{}, err
	}
	var spec Fixture
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&spec); err != nil {
		return LoadedFixture{}, fmt.Errorf("%w: fixture JSON is invalid: %v", producterror.ErrInvalidInput, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("trailing JSON value")
		}
		return LoadedFixture{}, fmt.Errorf("%w: fixture JSON has trailing data: %v", producterror.ErrInvalidInput, err)
	}
	loaded := LoadedFixture{Spec: spec, FixturePath: fixtureAbs, SHA256: bytesSHA256(raw)}
	if err := validateFixtureHeader(&loaded.Spec); err != nil {
		return LoadedFixture{}, err
	}
	parts, err := loadFixtureParts(archiveRoot, repoRoot, filepath.Dir(fixtureAbs), loaded.Spec.Parts)
	if err != nil {
		return LoadedFixture{}, err
	}
	loaded.Parts = parts
	return loaded, nil
}

func validateFixtureHeader(spec *Fixture) error {
	if strings.TrimSpace(spec.SchemaVersion) != FixtureSchemaVersion {
		return fmt.Errorf("%w: unsupported fixture schema", producterror.ErrInvalidInput)
	}
	if strings.TrimSpace(spec.FixtureID) == "" ||
		strings.TrimSpace(spec.SourceProvenance.ProvenanceID) == "" ||
		strings.TrimSpace(spec.SourceProvenance.ProductCommit) == "" ||
		strings.TrimSpace(spec.ReportTitle) == "" ||
		strings.TrimSpace(spec.Rigor.Level) == "" ||
		strings.TrimSpace(spec.Rigor.Label) == "" ||
		spec.WritingContract == nil {
		return fmt.Errorf("%w: fixture header is incomplete", producterror.ErrInvalidInput)
	}
	if spec.PostReportHumanize != strings.TrimSpace(spec.PostReportHumanize) {
		return fmt.Errorf("%w: fixture post_report_humanize must be canonical", producterror.ErrInvalidInput)
	}
	switch spec.PostReportHumanize {
	case reporting.FinalEditHumanizeEnabled, reporting.FinalEditHumanizeDisabled:
	default:
		return fmt.Errorf("%w: fixture post_report_humanize must be enabled or disabled", producterror.ErrInvalidInput)
	}
	if len(spec.Parts) == 0 {
		return fmt.Errorf("%w: fixture requires at least one Part", producterror.ErrInvalidInput)
	}
	return nil
}

func loadFixtureParts(archiveRoot, repoRoot, fixtureDir string, parts []FixturePart) ([]LoadedPart, error) {
	out := make([]LoadedPart, 0, len(parts))
	for index, part := range parts {
		if part.Index != index+1 || strings.TrimSpace(part.Title) == "" || strings.TrimSpace(part.Path) == "" || len(strings.TrimSpace(part.SHA256)) != 64 {
			return nil, fmt.Errorf("%w: fixture Part receipt is incomplete", producterror.ErrInvalidInput)
		}
		path := strings.TrimSpace(part.Path)
		if !filepath.IsAbs(path) {
			path = filepath.Join(fixtureDir, path)
		}
		abs, err := canonicalExistingPath(path)
		if err != nil {
			return nil, err
		}
		if !pathInside(archiveRoot, abs) {
			return nil, fmt.Errorf("%w: fixture Part path must be under archive root", producterror.ErrInvalidInput)
		}
		if err := rejectInsideRepo(repoRoot, abs); err != nil {
			return nil, err
		}
		content, err := os.ReadFile(abs)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(string(content)) == "" {
			return nil, fmt.Errorf("%w: fixture Part content is empty for index %d", producterror.ErrInvalidInput, part.Index)
		}
		sha := bytesSHA256(content)
		if !strings.EqualFold(sha, strings.TrimSpace(part.SHA256)) {
			return nil, fmt.Errorf("%w: fixture Part SHA-256 mismatch for index %d", producterror.ErrInvalidInput, part.Index)
		}
		out = append(out, LoadedPart{Spec: part, AbsPath: abs, SHA256: sha, Content: content, WordCount: len(strings.Fields(string(content)))})
	}
	return out, nil
}
