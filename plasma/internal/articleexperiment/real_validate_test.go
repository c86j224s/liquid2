package articleexperiment

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func TestLoadRealProtocolAcceptsMatchedProviderReadyBundle(t *testing.T) {
	archive, repo, protocolPath, protocolSHA := writeRealProtocolFixture(t, func(*RealProtocol, map[string]*RealArm, map[string]*RealFixture, *BlindContract) {})
	loaded, err := LoadRealProtocol(archive, repo, protocolPath, protocolSHA)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Protocol.ProtocolID != "real-pilot-v1" || loaded.FixtureCount() != 3 || loaded.ArmCount() != 3 || loaded.Blind.ContextualArm != ArmReport {
		t.Fatalf("loaded = %#v", loaded)
	}
}

func TestLoadRealProtocolRejectsUnmatchedEAAndUnsafeIsolation(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*RealProtocol, map[string]*RealArm, map[string]*RealFixture, *BlindContract)
	}{
		{name: "E/A provider differs", edit: func(_ *RealProtocol, arms map[string]*RealArm, _ map[string]*RealFixture, _ *BlindContract) {
			arms[ArmArticle].Provider.ReasoningEffort = "high"
		}},
		{name: "E/A common prompt differs", edit: func(_ *RealProtocol, arms map[string]*RealArm, _ map[string]*RealFixture, _ *BlindContract) {
			arms[ArmArticle].Common.ReaderPrompt = arms[ArmArticle].Treatment.Contract
		}},
		{name: "E/A treatment duplicated", edit: func(_ *RealProtocol, arms map[string]*RealArm, _ map[string]*RealFixture, _ *BlindContract) {
			arms[ArmArticle].Treatment.Contract = arms[ArmControl].Treatment.Contract
		}},
		{name: "ambient web enabled", edit: func(_ *RealProtocol, arms map[string]*RealArm, _ map[string]*RealFixture, _ *BlindContract) {
			arms[ArmArticle].Provider.AllowAmbientWeb = true
		}},
		{name: "unknown usage tolerated", edit: func(_ *RealProtocol, arms map[string]*RealArm, _ map[string]*RealFixture, _ *BlindContract) {
			arms[ArmArticle].Budget.UnknownUsagePolicy = "allow"
		}},
		{name: "raw provider log retained", edit: func(protocol *RealProtocol, _ map[string]*RealArm, _ map[string]*RealFixture, _ *BlindContract) {
			protocol.Isolation.RetainRawProviderLog = true
		}},
		{name: "invalid code revision", edit: func(protocol *RealProtocol, _ map[string]*RealArm, _ map[string]*RealFixture, _ *BlindContract) {
			protocol.CodeRevision = strings.Repeat("x", 40)
		}},
		{name: "R treatment changed", edit: func(_ *RealProtocol, arms map[string]*RealArm, _ map[string]*RealFixture, _ *BlindContract) {
			arms[ArmReport].Treatment.Kind = "article_narrative"
		}},
		{name: "R topology changed", edit: func(_ *RealProtocol, arms map[string]*RealArm, _ map[string]*RealFixture, _ *BlindContract) {
			arms[ArmReport].Stages[0].Stage = "other_product"
		}},
		{name: "artifact allowlist incomplete", edit: func(_ *RealProtocol, arms map[string]*RealArm, _ map[string]*RealFixture, _ *BlindContract) {
			arms[ArmControl].Artifacts = arms[ArmControl].Artifacts[:1]
			arms[ArmArticle].Artifacts = arms[ArmArticle].Artifacts[:1]
		}},
		{name: "token budget overflows", edit: func(_ *RealProtocol, arms map[string]*RealArm, _ map[string]*RealFixture, _ *BlindContract) {
			for _, armID := range requiredArmIDs {
				arms[armID].Budget.MaxInputTokens = math.MaxInt64
				arms[armID].Budget.MaxOutputTokens = math.MaxInt64
				arms[armID].Budget.MaxTotalTokens = 1
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			archive, repo, protocolPath, protocolSHA := writeRealProtocolFixture(t, test.edit)
			if _, err := LoadRealProtocol(archive, repo, protocolPath, protocolSHA); !errors.Is(err, producterror.ErrInvalidInput) {
				t.Fatalf("error = %v, want invalid input", err)
			}
		})
	}
}

func TestVerifyIssueLockReceiptBindsLiveCommentMetadataAndDigests(t *testing.T) {
	archive, repo, protocolPath, protocolSHA := writeRealProtocolFixture(t, func(*RealProtocol, map[string]*RealArm, map[string]*RealFixture, *BlindContract) {})
	loaded, err := LoadRealProtocol(archive, repo, protocolPath, protocolSHA)
	if err != nil {
		t.Fatal(err)
	}
	receipt := IssueLockReceipt{
		Repository: "c86j224s/liquid2", Issue: 461, Owner: "c86j224s",
		CommentURL: "https://github.com/c86j224s/liquid2/issues/461#issuecomment-123", CommentID: "123",
		CommentCreatedAt: "2026-09-03T01:00:00Z",
		ProtocolSHA256:   loaded.ProtocolSHA256, FixtureSHA256: loaded.FixtureDigests(),
		ArmSHA256: loaded.ArmDigests(), BlindSHA256: loaded.BlindSHA256,
	}
	body := []byte(strings.Join([]string{receipt.ProtocolSHA256, receipt.BlindSHA256, receipt.FixtureSHA256["M1"], receipt.FixtureSHA256["M2"], receipt.FixtureSHA256["M3"], receipt.ArmSHA256["R"], receipt.ArmSHA256["E"], receipt.ArmSHA256["A"]}, "\n") + "\n")
	receipt.CommentBodySHA256 = bytesSHA256(body)
	comment := LiveIssueComment{URL: receipt.CommentURL, ID: receipt.CommentID, Author: "c86j224s", CreatedAt: receipt.CommentCreatedAt, Body: body}
	if err := VerifyIssueLockReceipt(loaded, receipt, comment); err != nil {
		t.Fatal(err)
	}
	comment.Author = "other-user"
	if err := VerifyIssueLockReceipt(loaded, receipt, comment); !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("non-owner error = %v, want conflict", err)
	}
	comment.Author = "c86j224s"
	comment.ID = "456"
	if err := VerifyIssueLockReceipt(loaded, receipt, comment); !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("comment identity error = %v, want conflict", err)
	}
	comment.ID = receipt.CommentID
	receipt.ProtocolSHA256 = strings.Repeat("0", 64)
	if err := VerifyIssueLockReceipt(loaded, receipt, comment); !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("error = %v, want conflict", err)
	}
}

func TestLoadRealProtocolRejectsBlindAndFixtureDrift(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*RealProtocol, map[string]*RealArm, map[string]*RealFixture, *BlindContract)
	}{
		{name: "R enters primary lane", edit: func(_ *RealProtocol, _ map[string]*RealArm, _ map[string]*RealFixture, blind *BlindContract) {
			blind.PrimaryArms = []string{ArmReport, ArmArticle}
		}},
		{name: "generation order duplicates", edit: func(_ *RealProtocol, _ map[string]*RealArm, _ map[string]*RealFixture, blind *BlindContract) {
			blind.GenerationOrder[8] = blind.GenerationOrder[0]
		}},
		{name: "missing arm guess", edit: func(_ *RealProtocol, _ map[string]*RealArm, _ map[string]*RealFixture, blind *BlindContract) {
			blind.ArmGuessQuestion = ""
		}},
		{name: "missing randomization seed", edit: func(_ *RealProtocol, _ map[string]*RealArm, _ map[string]*RealFixture, blind *BlindContract) {
			blind.RandomizationSeedSHA256 = ""
		}},
		{name: "source claim count too small", edit: func(_ *RealProtocol, _ map[string]*RealArm, fixtures map[string]*RealFixture, _ *BlindContract) {
			fixtures["M2"].MaterialTruthClaims = 7
		}},
		{name: "body band changed", edit: func(_ *RealProtocol, _ map[string]*RealArm, fixtures map[string]*RealFixture, _ *BlindContract) {
			fixtures["M3"].MaximumBodyCharacters = 12000
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			archive, repo, protocolPath, protocolSHA := writeRealProtocolFixture(t, test.edit)
			if _, err := LoadRealProtocol(archive, repo, protocolPath, protocolSHA); !errors.Is(err, producterror.ErrInvalidInput) {
				t.Fatalf("error = %v, want invalid input", err)
			}
		})
	}
}

func writeMinimalFixtureArtifacts(t *testing.T, archive, fixtureID string) []FileRef {
	t.Helper()
	dir := filepath.Join(archive, "fixtures")
	body := []byte(strings.Repeat("source fact\n", 10))
	bodyPath := filepath.Join(dir, fixtureID+"-body.txt")
	if err := os.WriteFile(bodyPath, body, 0o600); err != nil {
		t.Fatal(err)
	}
	rightsPath := filepath.Join(dir, fixtureID+"-rights.txt")
	if err := os.WriteFile(rightsPath, []byte("rights\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	headersPath := filepath.Join(dir, fixtureID+"-headers.txt")
	if err := os.WriteFile(headersPath, []byte("HTTP/2 200\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	urlPath := filepath.Join(dir, fixtureID+"-url.txt")
	if err := os.WriteFile(urlPath, []byte("https://example.invalid/source\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	spans := []FixtureSourceSpan{}
	claims := []FixtureClaim{}
	accounts := []FixtureAccount{}
	for i := 0; i < 10; i++ {
		start := i * len("source fact\n")
		end := start + len("source fact")
		sid := fmt.Sprintf("%s-span-%02d", strings.ToLower(fixtureID), i)
		cid := fmt.Sprintf("%s-claim-%02d", strings.ToLower(fixtureID), i)
		spans = append(spans, FixtureSourceSpan{SpanID: sid, SourceKey: "source_001", StartByte: start, EndByte: end, SHA256: bytesSHA256(body[start:end])})
		claims = append(claims, FixtureClaim{ClaimID: cid, SummaryKO: "근거 사실", SupportRelation: "direct_or_bounded", Uncertainty: "preserve source qualifiers", ClaimStrength: "material", SpanIDs: []string{sid}})
		accounts = append(accounts, FixtureAccount{AccountID: fmt.Sprintf("%s-account-%02d", strings.ToLower(fixtureID), i), Kind: "factual_relationship", ClaimIDs: []string{cid}, SpanIDs: []string{sid}})
	}
	ref := func(id, path string) FileRef {
		return FileRef{ID: id, Path: filepath.Base(path), SHA256: testFileSHA256(t, path)}
	}
	catalog := FixtureSourceCatalog{SchemaVersion: SourceCatalogSchemaVersion, FixtureID: fixtureID, Sources: []FixtureSource{{SourceKey: "source_001", Content: ref(fixtureID+"-body", bodyPath), MediaType: "text/markdown; charset=utf-8", RetrievalPolicy: "snapshot_only", Extraction: "test", Derivation: []SourceDerivation{{OriginID: strings.ToLower(fixtureID) + "-origin", OriginSpan: "source fact", SHA256: bytesSHA256([]byte("source fact"))}}, MediaExcluded: true, Rights: FixtureRights{Agency: "test", RightsURL: "https://example.invalid/rights", AIUse: "test"}, Origins: []FixtureOrigin{{OriginID: strings.ToLower(fixtureID) + "-origin", URL: "https://example.invalid/source", Raw: ref(fixtureID+"-raw", bodyPath), Headers: ref(fixtureID+"-headers", headersPath), FinalURL: ref(fixtureID+"-url", urlPath), Rights: ref(fixtureID+"-rights", rightsPath)}}}}, Spans: spans}
	dossier := FixtureDossier{SchemaVersion: DossierSchemaVersion, FixtureID: fixtureID, Sensitive: true, Accounts: accounts, Connections: []FixtureConnection{{ConnectionID: strings.ToLower(fixtureID) + "-connection", Kind: "relationship", FromClaimIDs: []string{claims[0].ClaimID}, ToClaimIDs: []string{claims[1].ClaimID}, SummaryKO: "연결"}}, Caveats: []FixtureCaveat{{CaveatID: strings.ToLower(fixtureID) + "-caveat-1", SummaryKO: "주의", SpanIDs: []string{spans[0].SpanID}}, {CaveatID: strings.ToLower(fixtureID) + "-caveat-2", SummaryKO: "제한", SpanIDs: []string{spans[1].SpanID}}}, Forbidden: []string{"invented fact"}}
	inventory := FixtureClaimInventory{SchemaVersion: ClaimInventorySchemaVersion, FixtureID: fixtureID, Claims: claims, RequiredLeafCoverage: "all authored strings", Verdicts: []string{"directly_supported", "bounded_inference", "overstated", "contradicted", "unsupported"}}
	memory := FixtureMemoryQuestions{SchemaVersion: MemoryQuestionsSchemaVersion, FixtureID: fixtureID, Questions: []FixtureMemoryQuestion{{QuestionID: strings.ToLower(fixtureID) + "-memory", QuestionKO: "무엇인가?", RequiredClaimIDs: []string{claims[0].ClaimID}, Scoring: "0-2"}}}
	transfer := FixtureTransferTask{SchemaVersion: TransferTaskSchemaVersion, FixtureID: fixtureID, TaskKO: "적용하라", RequiredClaimIDs: []string{claims[1].ClaimID}, Scoring: "0-2"}
	values := []struct {
		name  string
		value any
	}{{fixtureID + "-source.json", catalog}, {fixtureID + "-dossier.json", dossier}, {fixtureID + "-claims.json", inventory}, {fixtureID + "-memory.json", memory}, {fixtureID + "-transfer.json", transfer}}
	refs := []FileRef{}
	for _, v := range values {
		path := filepath.Join(dir, v.name)
		raw := writeRealJSON(t, path, v.value)
		refs = append(refs, FileRef{ID: strings.TrimSuffix(v.name, ".json"), Path: v.name, SHA256: bytesSHA256(raw)})
	}
	return refs
}

func writeRealProtocolFixture(t *testing.T, edit func(*RealProtocol, map[string]*RealArm, map[string]*RealFixture, *BlindContract)) (string, string, string, string) {
	t.Helper()
	archive := t.TempDir()
	if err := os.Chmod(archive, 0o700); err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	for _, directory := range []string{"fixtures", "arms", "contracts", "blind"} {
		if err := os.Mkdir(filepath.Join(archive, directory), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	contractRef := func(id, ownerDir string) FileRef {
		path := filepath.Join(archive, ownerDir, id+".txt")
		content := []byte(id + "\n")
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatal(err)
		}
		return FileRef{ID: id, Path: id + ".txt", SHA256: bytesSHA256(content)}
	}
	fixtureMap := map[string]*RealFixture{}
	fixtureRefs := make([]FileRef, 0, 3)
	for _, fixtureID := range requiredFixtureIDs {
		artifactRefs := writeMinimalFixtureArtifacts(t, archive, fixtureID)
		fixture := &RealFixture{
			SchemaVersion: RealFixtureSchemaVersion, FixtureID: fixtureID,
			ValueType: map[string]string{"M1": "method_how_to", "M2": "mechanism_insight", "M3": "discovery_context"}[fixtureID],
			Audience:  "처음 읽는 독자", ReaderPromise: "읽고 핵심을 이해한다", Language: "ko",
			TargetBodyCharacters: 9000, MinimumBodyCharacters: 7000, MaximumBodyCharacters: 11000,
			MaterialTruthClaims: 10, MaterialCaveats: 2, SupportedConnections: 1,
			SourceCatalog: artifactRefs[0], Dossier: artifactRefs[1], ClaimInventory: artifactRefs[2],
			MemoryQuestions: artifactRefs[3], TransferTask: artifactRefs[4],
		}
		fixtureMap[fixtureID] = fixture
	}
	provider := ProviderContract{Executor: pilotProvider, Model: pilotModel, ReasoningEffort: pilotReasoningEffort, FreshEphemeralSessions: true, IgnoreUserConfig: true, ReplaceMCPTools: true}
	stages := make([]StageContract, 0, len(requiredRealStages))
	for _, stage := range requiredRealStages {
		access := "read_only"
		if stage == "author" || stage == "conditional_author_repair" || stage == "render" {
			access = "author_or_server"
		}
		stages = append(stages, StageContract{Stage: stage, Access: access, MaxAttempts: 1})
	}
	common := ArmCommonContract{AuthorPrompt: contractRef("author-prompt", "arms"), ReaderPrompt: contractRef("reader-prompt", "arms"), AuditorPrompt: contractRef("auditor-prompt", "arms"), RepairPrompt: contractRef("repair-prompt", "arms"), ToolPolicy: contractRef("tool-policy", "arms"), OutputSchema: contractRef("output-schema", "arms"), Renderer: contractRef("renderer", "arms")}
	budget := CellBudget{MaxProviderCalls: 8, MaxTechnicalRetries: 1, MaxRepairBatches: 1, MaxDurationMilliseconds: 3600000, MaxSourceBytesPerCall: 64 << 10, MaxSourceBytesPerAttempt: 512 << 10, MaxTotalTokens: 1000000, MaxInputTokens: 800000, MaxOutputTokens: 200000, UnknownUsagePolicy: "fail_closed"}
	artifacts := []ArtifactContract{
		{Kind: "article_il", MediaType: "application/json", Filename: "article-il.json", MaxByteSize: 16 << 20, RequireUTF8: true},
		{Kind: "article_markdown", MediaType: "text/markdown; charset=utf-8", Filename: "article.md", MaxByteSize: 16 << 20, RequireUTF8: true},
		{Kind: "article_html", MediaType: "text/html; charset=utf-8", Filename: "article.html", MaxByteSize: 16 << 20, RequireUTF8: true},
		{Kind: "article_pdf", MediaType: "application/pdf", Filename: "article.pdf", MaxByteSize: 16 << 20},
		{Kind: "usage_receipt", MediaType: "application/json", Filename: "usage.json", MaxByteSize: 1 << 20, RequireUTF8: true},
		{Kind: "audit_receipt", MediaType: "application/json", Filename: "audit.json", MaxByteSize: 1 << 20, RequireUTF8: true},
	}
	reportArtifacts := []ArtifactContract{
		{Kind: "report_markdown", MediaType: "text/markdown; charset=utf-8", Filename: "report.md", MaxByteSize: 16 << 20, RequireUTF8: true},
		{Kind: "report_html", MediaType: "text/html; charset=utf-8", Filename: "report.html", MaxByteSize: 16 << 20, RequireUTF8: true},
		{Kind: "report_pdf", MediaType: "application/pdf", Filename: "report.pdf", MaxByteSize: 16 << 20},
		{Kind: "usage_receipt", MediaType: "application/json", Filename: "usage.json", MaxByteSize: 1 << 20, RequireUTF8: true},
		{Kind: "audit_receipt", MediaType: "application/json", Filename: "audit.json", MaxByteSize: 1 << 20, RequireUTF8: true},
	}
	armMap := map[string]*RealArm{
		ArmReport:  {SchemaVersion: RealArmSchemaVersion, ArmID: ArmReport, Role: "contextual_report_il", ContextualOnly: true, Provider: provider, Stages: []StageContract{{Stage: "report_product", Access: "product_or_server", MaxAttempts: 1}}, Common: common, Treatment: TreatmentContract{Kind: "report_product", Contract: contractRef("treatment-R", "arms")}, Budget: budget, Artifacts: reportArtifacts},
		ArmControl: {SchemaVersion: RealArmSchemaVersion, ArmID: ArmControl, Role: "matched_control", Provider: provider, Stages: append([]StageContract(nil), stages...), Common: common, Treatment: TreatmentContract{Kind: "none", Contract: contractRef("treatment-E", "arms")}, Budget: budget, Artifacts: artifacts},
		ArmArticle: {SchemaVersion: RealArmSchemaVersion, ArmID: ArmArticle, Role: "article_narrative", Provider: provider, Stages: append([]StageContract(nil), stages...), Common: common, Treatment: TreatmentContract{Kind: "article_narrative", Contract: contractRef("treatment-A", "arms")}, Budget: budget, Artifacts: artifacts},
	}
	cells := make([]RunCell, 0, 9)
	for _, fixtureID := range requiredFixtureIDs {
		for _, armID := range requiredArmIDs {
			cells = append(cells, RunCell{FixtureID: fixtureID, ArmID: armID, RunID: "real-" + fixtureID + "-" + armID})
		}
	}
	blind := BlindContract{SchemaVersion: BlindSchemaVersion, PrimaryArms: []string{ArmControl, ArmArticle}, ContextualArm: ArmReport, MappingCommitmentSHA256: bytesSHA256([]byte("private mapping")), RandomizationSeedSHA256: bytesSHA256([]byte("private seed")), GenerationOrder: []string{"real-M1-E", "real-M2-A", "real-M3-R", "real-M1-A", "real-M2-R", "real-M3-E", "real-M1-R", "real-M2-E", "real-M3-A"}, Pairs: []BlindPair{{FixtureID: "M1", PacketID: "pair-M1", Slots: []string{"slot-1", "slot-2"}}, {FixtureID: "M2", PacketID: "pair-M2", Slots: []string{"slot-1", "slot-2"}}, {FixtureID: "M3", PacketID: "pair-M3", Slots: []string{"slot-1", "slot-2"}}}, Rubric: contractRef("rubric", "blind"), ReaderShell: contractRef("reader-shell", "blind"), Administration: contractRef("administration", "blind"), ArmGuessQuestion: "어느 슬롯이 Article Narrative라고 생각하십니까?", RevealAfterUserJudgment: true, RevealAfterFactualAudits: true, NativeLaneAfterReveal: true}
	protocol := RealProtocol{SchemaVersion: RealProtocolSchemaVersion, ProtocolID: "real-pilot-v1", CreatedAt: "2026-09-03T00:00:00Z", CodeRevision: "7443ff68518891bc00a5e607e9340f36783961b5", IssueLock: IssueLockContract{Repository: "c86j224s/liquid2", Issue: 461, Owner: "c86j224s", Required: true}, Runs: cells, Isolation: IsolationContract{ArchiveMode: "0700", FileMode: "0600", DedicatedArchive: true, SingleWriter: true, IsolatedHome: true, IsolatedProviderConfig: true, IsolatedWorkspace: true, IsolatedDatabase: true, EnvironmentAllowlist: true}, MatrixBudget: MatrixBudget{MaxCells: 9, MaxProviderCalls: 72, MaxWallTimeMilliseconds: 8 * 3600000, MaxTotalTokens: 9000000, MaxInputTokens: 7200000, MaxOutputTokens: 1800000, OnCeiling: "stop_new_cells", UnknownUsagePolicy: "fail_closed", BudgetAccountingVersion: "agentusage.increment.v1"}, FailurePolicy: RealFailurePolicy{SemanticAttemptsPerCell: 1, AnyUnusableCellOutcome: "INCONCLUSIVE", ProtocolChangeOutcome: "INVALID"}}
	edit(&protocol, armMap, fixtureMap, &blind)
	for _, fixtureID := range requiredFixtureIDs {
		path := filepath.Join(archive, "fixtures", fixtureID+".json")
		raw := writeRealJSON(t, path, fixtureMap[fixtureID])
		fixtureRefs = append(fixtureRefs, FileRef{ID: fixtureID, Path: filepath.ToSlash(filepath.Join("fixtures", fixtureID+".json")), SHA256: bytesSHA256(raw)})
	}
	armRefs := make([]FileRef, 0, 3)
	for _, armID := range requiredArmIDs {
		path := filepath.Join(archive, "arms", armID+".json")
		raw := writeRealJSON(t, path, armMap[armID])
		armRefs = append(armRefs, FileRef{ID: armID, Path: filepath.ToSlash(filepath.Join("arms", armID+".json")), SHA256: bytesSHA256(raw)})
	}
	blindPath := filepath.Join(archive, "blind", "blind.json")
	blindRaw := writeRealJSON(t, blindPath, blind)
	protocol.Fixtures = fixtureRefs
	protocol.Arms = armRefs
	protocol.Blind = FileRef{ID: "blind", Path: filepath.ToSlash(filepath.Join("blind", "blind.json")), SHA256: bytesSHA256(blindRaw)}
	protocolPath := filepath.Join(archive, "protocol.json")
	protocolRaw := writeRealJSON(t, protocolPath, protocol)
	return archive, repo, protocolPath, bytesSHA256(protocolRaw)
}

func writeRealJSON(t *testing.T, path string, value any) []byte {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return raw
}
