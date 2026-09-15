package articlepilot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/articleexperiment"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

const maxPromptBytes = 256 << 10

// Provider is the existing neutral process boundary used by the minimal pilot.
type Provider interface {
	Run(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error)
}

// Config freezes one E/A cell. Prompts are loaded from the already-hashed arm
// contract by the command adapter before this runner is constructed.
type Config struct {
	ProtocolSHA256 string
	Bundle         articleexperiment.FixtureBundle
	Arm            articleexperiment.RealArm
	AuthorPrompt   string
	ReaderPrompt   string
	AuditorPrompt  string
	RepairPrompt   string
	Treatment      []byte
	Provider       Provider
	Now            func() time.Time
}

type Result struct {
	Document   Document
	Manuscript []byte
	Usage      UsageReceipt
	Audit      AuditReceipt
}

type UsageReceipt struct {
	ProtocolSHA256 string        `json:"protocol_sha256"`
	FixtureID      string        `json:"fixture_id"`
	ArmID          string        `json:"arm_id"`
	Calls          []CallReceipt `json:"calls"`
	InputTokens    int64         `json:"input_tokens"`
	OutputTokens   int64         `json:"output_tokens"`
	TotalTokens    int64         `json:"total_tokens"`
	DurationMS     int64         `json:"duration_ms"`
}

type CallReceipt struct {
	Stage        string `json:"stage"`
	PromptSHA256 string `json:"prompt_sha256"`
	ResultSHA256 string `json:"result_sha256"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	TotalTokens  int    `json:"total_tokens"`
	DurationMS   int64  `json:"duration_ms"`
}

type AuditReceipt struct {
	ReaderNeedsRepair  bool `json:"reader_needs_repair"`
	FactualNeedsRepair bool `json:"factual_needs_repair"`
	RepairApplied      bool `json:"repair_applied"`
	ConfirmationPassed bool `json:"confirmation_passed"`
}

type diagnosis struct {
	NeedsRepair bool     `json:"needs_repair"`
	Findings    []string `json:"findings"`
}

type repairBatch struct {
	Replacements []replacement `json:"replacements"`
}

type replacement struct {
	NodeID  string `json:"node_id"`
	OldText string `json:"old_text"`
	NewText string `json:"new_text"`
}

func Run(ctx context.Context, config Config) (Result, error) {
	if config.Provider == nil || config.Bundle.Fixture.FixtureID == "" || (config.Arm.ArmID != articleexperiment.ArmControl && config.Arm.ArmID != articleexperiment.ArmArticle) || len(config.Treatment) == 0 || !utf8.Valid(config.Treatment) {
		return Result{}, fmt.Errorf("%w: minimal Article pilot config is invalid", producterror.ErrInvalidInput)
	}
	if err := validatePromptInputs(config); err != nil {
		return Result{}, err
	}
	sourceBytes := int64(len(config.Bundle.SourceBody) + len(config.Bundle.SourceCatalog) + len(config.Bundle.Dossier) + len(config.Bundle.ClaimInventory))
	budget := config.Arm.Budget
	if sourceBytes > budget.MaxSourceBytesPerCall || sourceBytes > budget.MaxSourceBytesPerAttempt/4 {
		return Result{}, fmt.Errorf("%w: Article pilot source bytes exceed ceiling", producterror.ErrInvalidInput)
	}
	usage := UsageReceipt{ProtocolSHA256: config.ProtocolSHA256, FixtureID: config.Bundle.Fixture.FixtureID, ArmID: config.Arm.ArmID}
	usageBySession := map[string]agentusage.AgentUsage{}
	authorPrompt := authorInput(config)
	author, err := runProvider(ctx, config, "author", authorPrompt, "", &usage, usageBySession)
	if err != nil {
		return Result{Usage: usage}, err
	}
	if author.Resumed || strings.TrimSpace(author.SessionID) == "" {
		return Result{Usage: usage}, fmt.Errorf("%w: author session is not fresh", producterror.ErrConflict)
	}
	document, err := ParseDocument([]byte(author.Text), config.Bundle.Fixture, config.Bundle.ClaimIDs)
	if err != nil {
		return Result{Usage: usage}, err
	}
	manuscript := RenderMarkdown(document)
	reader, err := runDiagnosis(ctx, config, "reader", config.ReaderPrompt, manuscript, &usage, usageBySession)
	if err != nil {
		return Result{Document: document, Manuscript: manuscript, Usage: usage}, err
	}
	auditorInstruction := config.AuditorPrompt + "\n\nSOURCE:\n" + string(config.Bundle.SourceBody) + "\n\nSOURCE CATALOG:\n" + string(config.Bundle.SourceCatalog) + "\n\nDOSSIER:\n" + string(config.Bundle.Dossier) + "\n\nCLAIM INVENTORY:\n" + string(config.Bundle.ClaimInventory)
	auditor, err := runDiagnosis(ctx, config, "factual_audit", auditorInstruction, manuscript, &usage, usageBySession)
	if err != nil {
		return Result{Document: document, Manuscript: manuscript, Usage: usage}, err
	}
	audit := AuditReceipt{ReaderNeedsRepair: reader.NeedsRepair, FactualNeedsRepair: auditor.NeedsRepair}
	if reader.NeedsRepair || auditor.NeedsRepair {
		repairPrompt := config.RepairPrompt + "\n\nREADER FINDINGS:\n" + joinFindings(reader.Findings) + "\n\nFACTUAL FINDINGS:\n" + joinFindings(auditor.Findings) + "\n\nMANUSCRIPT:\n" + string(manuscript)
		repair, err := runProvider(ctx, config, "repair", repairPrompt, author.SessionID, &usage, usageBySession)
		if err != nil {
			return Result{Document: document, Manuscript: manuscript, Usage: usage, Audit: audit}, err
		}
		if !repair.Resumed || repair.SessionID != author.SessionID {
			return Result{Document: document, Manuscript: manuscript, Usage: usage, Audit: audit}, fmt.Errorf("%w: author repair lineage differs", producterror.ErrConflict)
		}
		var batch repairBatch
		if err := decodeStrict([]byte(repair.Text), &batch); err != nil {
			return Result{Document: document, Manuscript: manuscript, Usage: usage, Audit: audit}, err
		}
		document, err = applyDocumentRepair(document, batch)
		if err != nil {
			return Result{Document: document, Manuscript: manuscript, Usage: usage, Audit: audit}, err
		}
		documentRaw, err := json.Marshal(document)
		if err != nil {
			return Result{Document: document, Manuscript: manuscript, Usage: usage, Audit: audit}, err
		}
		document, err = ParseDocument(documentRaw, config.Bundle.Fixture, config.Bundle.ClaimIDs)
		if err != nil {
			return Result{Document: document, Manuscript: manuscript, Usage: usage, Audit: audit}, err
		}
		manuscript = RenderMarkdown(document)
		audit.RepairApplied = true
		readerConfirmation, err := runDiagnosis(ctx, config, "reader_confirmation", config.ReaderPrompt+"\nConfirm the repaired manuscript has no remaining reader finding.", manuscript, &usage, usageBySession)
		if err != nil {
			return Result{Document: document, Manuscript: manuscript, Usage: usage, Audit: audit}, err
		}
		factualConfirmation, err := runDiagnosis(ctx, config, "factual_confirmation", auditorInstruction+"\nConfirm the repaired manuscript has no remaining material factual finding.", manuscript, &usage, usageBySession)
		if err != nil {
			return Result{Document: document, Manuscript: manuscript, Usage: usage, Audit: audit}, err
		}
		if readerConfirmation.NeedsRepair || factualConfirmation.NeedsRepair {
			return Result{Document: document, Manuscript: manuscript, Usage: usage, Audit: audit}, fmt.Errorf("%w: Article repair was exhausted", producterror.ErrConflict)
		}
		audit.ConfirmationPassed = true
	} else {
		audit.ConfirmationPassed = true
	}
	return Result{Document: document, Manuscript: manuscript, Usage: usage, Audit: audit}, nil
}

func runDiagnosis(ctx context.Context, config Config, stage, instruction string, manuscript []byte, usage *UsageReceipt, usageBySession map[string]agentusage.AgentUsage) (diagnosis, error) {
	prompt := instruction + "\n\nReturn strict JSON with needs_repair and findings. Do not rewrite prose.\n\nMANUSCRIPT:\n" + string(manuscript)
	result, err := runProvider(ctx, config, stage, prompt, "", usage, usageBySession)
	if err != nil {
		return diagnosis{}, err
	}
	if result.Resumed || strings.TrimSpace(result.SessionID) == "" {
		return diagnosis{}, fmt.Errorf("%w: independent reviewer session is not fresh", producterror.ErrConflict)
	}
	var value diagnosis
	if err := decodeStrict([]byte(result.Text), &value); err != nil {
		return diagnosis{}, err
	}
	if value.NeedsRepair != (len(value.Findings) > 0) || len(value.Findings) > 6 {
		return diagnosis{}, fmt.Errorf("%w: diagnosis shape is invalid", producterror.ErrInvalidInput)
	}
	return value, nil
}

func runProvider(ctx context.Context, config Config, stage, prompt, previousSession string, usage *UsageReceipt, usageBySession map[string]agentusage.AgentUsage) (agentexec.AgentResult, error) {
	if len([]byte(prompt)) > maxPromptBytes {
		return agentexec.AgentResult{}, fmt.Errorf("%w: Article pilot prompt exceeds ceiling", producterror.ErrInvalidInput)
	}
	budget := config.Arm.Budget
	if len(usage.Calls) >= budget.MaxProviderCalls {
		return agentexec.AgentResult{}, fmt.Errorf("%w: Article pilot provider-call budget exhausted", producterror.ErrConflict)
	}
	remaining := budget.MaxDurationMilliseconds - usage.DurationMS
	if remaining <= 0 {
		return agentexec.AgentResult{}, fmt.Errorf("%w: Article pilot duration budget exhausted", producterror.ErrConflict)
	}
	started := now(config)
	callCtx, cancel := context.WithTimeout(ctx, time.Duration(remaining)*time.Millisecond)
	defer cancel()
	result, err := config.Provider.Run(callCtx, agentexec.AgentRequest{Prompt: prompt, Model: config.Arm.Provider.Model, ReasoningEffort: config.Arm.Provider.ReasoningEffort, PreviousSessionID: previousSession, AgentExecutor: "codex", DisableTools: true, IgnoreUserConfig: true, ReplaceMCPTools: true, EphemeralSession: previousSession == "" && stage != "author", PreserveResponseWhitespace: true})
	duration := now(config).Sub(started).Milliseconds()
	if errors.Is(err, context.Canceled) {
		return result, context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return result, context.DeadlineExceeded
	}
	if err != nil {
		return result, fmt.Errorf("%w: Article pilot provider failed", producterror.ErrConflict)
	}
	var previousUsage *agentusage.AgentUsage
	if previousSession != "" {
		if prior, ok := usageBySession[previousSession]; ok {
			priorCopy := prior
			previousUsage = &priorCopy
		}
	}
	increment, _, ok := agentusage.IncrementalProviderUsage(result.Usage, previousUsage)
	if !ok || result.Usage.UsageUnavailable {
		return result, fmt.Errorf("%w: provider usage is unavailable", producterror.ErrConflict)
	}
	if sessionID := strings.TrimSpace(result.Usage.Session.AgentSessionID); sessionID != "" {
		usageBySession[sessionID] = result.Usage
	}
	call := CallReceipt{Stage: stage, PromptSHA256: digest([]byte(prompt)), ResultSHA256: digest([]byte(result.Text)), InputTokens: increment.InputTokens, OutputTokens: increment.OutputTokens, TotalTokens: increment.TotalTokens, DurationMS: duration}
	usage.Calls = append(usage.Calls, call)
	usage.InputTokens += int64(increment.InputTokens)
	usage.OutputTokens += int64(increment.OutputTokens)
	usage.TotalTokens += int64(increment.TotalTokens)
	usage.DurationMS += duration
	if usage.InputTokens > budget.MaxInputTokens || usage.OutputTokens > budget.MaxOutputTokens || usage.TotalTokens > budget.MaxTotalTokens || usage.DurationMS > budget.MaxDurationMilliseconds {
		return result, fmt.Errorf("%w: Article pilot budget exceeded", producterror.ErrConflict)
	}
	return result, nil
}

func authorInput(config Config) string {
	return config.AuthorPrompt + "\n\nTREATMENT CONTRACT:\n" + string(config.Treatment) + "\n\nAUDIENCE:\n" + config.Bundle.Fixture.Audience + "\n\nREADER PROMISE:\n" + config.Bundle.Fixture.ReaderPromise + "\n\nEMPHASIS:\n" + config.Bundle.Fixture.Emphasis + "\n\nSOURCE:\n" + string(config.Bundle.SourceBody) + "\n\nDOSSIER:\n" + string(config.Bundle.Dossier) + "\n\nCLAIMS:\n" + string(config.Bundle.ClaimInventory)
}

func validatePromptInputs(config Config) error {
	for _, value := range []string{config.AuthorPrompt, config.ReaderPrompt, config.AuditorPrompt, config.RepairPrompt} {
		if strings.TrimSpace(value) == "" || !utf8.ValidString(value) {
			return fmt.Errorf("%w: Article pilot prompt is invalid", producterror.ErrInvalidInput)
		}
	}
	return nil
}

func applyDocumentRepair(document Document, batch repairBatch) (Document, error) {
	if len(batch.Replacements) == 0 || len(batch.Replacements) > 6 {
		return Document{}, fmt.Errorf("%w: repair batch size is invalid", producterror.ErrInvalidInput)
	}
	totalNew := 0
	seenNodes := map[string]bool{}
	for _, operation := range batch.Replacements {
		leaf, ok := documentLeaf(&document, operation.NodeID)
		if !ok || seenNodes[operation.NodeID] || operation.OldText == "" || operation.NewText == "" || len([]byte(operation.OldText)) > 1024 || len([]byte(operation.NewText)) > 1024 || strings.Count(leaf.Text, operation.OldText) != 1 {
			return Document{}, fmt.Errorf("%w: repair replacement is invalid", producterror.ErrInvalidInput)
		}
		seenNodes[operation.NodeID] = true
		oldNode := []rune(leaf.Text)
		oldText, newText := []rune(operation.OldText), []rune(operation.NewText)
		newNode := []rune(strings.Replace(leaf.Text, operation.OldText, operation.NewText, 1))
		if len(oldText)*4 > len(oldNode) || runeDistance(oldText, newText)*4 > len(oldNode) || len(newNode)*100 < len(oldNode)*85 || len(newNode)*100 > len(oldNode)*115 {
			return Document{}, fmt.Errorf("%w: repair exceeds node-local edit boundary", producterror.ErrInvalidInput)
		}
		totalNew += len([]byte(operation.NewText))
		if totalNew > 4096 {
			return Document{}, fmt.Errorf("%w: repair text exceeds ceiling", producterror.ErrInvalidInput)
		}
		leaf.Text = string(newNode)
	}
	return document, nil
}

func runeDistance(left, right []rune) int {
	previous := make([]int, len(right)+1)
	for index := range previous {
		previous[index] = index
	}
	for i, leftRune := range left {
		current := make([]int, len(right)+1)
		current[0] = i + 1
		for j, rightRune := range right {
			cost := 0
			if leftRune != rightRune {
				cost = 1
			}
			current[j+1] = min(current[j]+1, previous[j+1]+1, previous[j]+cost)
		}
		previous = current
	}
	return previous[len(right)]
}

func joinFindings(values []string) string {
	return strings.Join(values, "\n- ")
}

func now(config Config) time.Time {
	if config.Now != nil {
		return config.Now()
	}
	return time.Now()
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}
