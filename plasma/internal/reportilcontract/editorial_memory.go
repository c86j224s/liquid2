package reportilcontract

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	EditorialMemorySchemaVersion  = "plasma.report_il.editorial_memory.experimental.v2"
	EditorialMemoryMediaType      = "application/vnd.plasma.report-il-editorial-memory+json"
	MaxEditorialMemoryBytes       = 256 * 1024
	MaxEditorialAccounts          = 128
	MaxEditorialAnchorsPerAccount = 16
)

var editorialAccountKeyPattern = regexp.MustCompile(`^account_[0-9]{3}$`)

// EditorialMemory preserves material source accounts before prose compression.
// It is an authoring input, not a reader-facing outline or an audit artifact.
type EditorialMemory struct {
	SchemaVersion string             `json:"schema_version"`
	Language      string             `json:"language"`
	Accounts      []EditorialAccount `json:"accounts"`
}

type EditorialAccount struct {
	AccountKey string   `json:"account_key"`
	Importance string   `json:"importance"`
	Account    string   `json:"account"`
	SourceKeys []string `json:"source_keys"`
}

// EditorialAnchor preserves a bounded exact excerpt from one frozen source.
// It is reconstructed and verified by the server; the provider cannot supply
// offsets or hashes directly.
type EditorialAnchor struct {
	AccountKey string `json:"account_key"`
	SourceKey  string `json:"source_key"`
	Excerpt    string `json:"excerpt"`
	Offset     int    `json:"offset"`
	ByteSize   int    `json:"byte_size"`
	SHA256     string `json:"sha256"`
}

// EditorialMemoryArtifact is the server-owned downstream view. Memory keeps
// semantic account validation independent from storage-bound exact excerpts.
type EditorialMemoryArtifact struct {
	SchemaVersion string             `json:"schema_version"`
	Language      string             `json:"language"`
	Accounts      []EditorialAccount `json:"accounts"`
	Anchors       []EditorialAnchor  `json:"anchors"`
}

type EditorialMemoryReceipt struct {
	WorkspaceID  string
	ArtifactID   string
	SHA256       string
	ByteSize     int
	Revision     int
	Accounts     int
	Replacements int
}

func ValidateEditorialMemory(memory EditorialMemory, catalog SourceCatalog) error {
	if err := ValidateSourceCatalog(catalog); err != nil {
		return err
	}
	if memory.SchemaVersion != EditorialMemorySchemaVersion ||
		!authorDocumentLanguagePattern.MatchString(memory.Language) ||
		len(memory.Accounts) == 0 || len(memory.Accounts) > MaxEditorialAccounts {
		return fmt.Errorf("editorial memory envelope is invalid")
	}
	seenAccounts := make(map[string]bool, len(memory.Accounts))
	for index, account := range memory.Accounts {
		wantKey := fmt.Sprintf("account_%03d", index+1)
		if account.AccountKey != wantKey || !editorialAccountKeyPattern.MatchString(account.AccountKey) ||
			seenAccounts[account.AccountKey] ||
			(account.Importance != "essential" && account.Importance != "supporting") ||
			strings.TrimSpace(account.Account) == "" ||
			!utf8.ValidString(account.Account) || len(account.SourceKeys) == 0 {
			return fmt.Errorf("editorial memory account is invalid")
		}
		seenAccounts[account.AccountKey] = true
		seenSources := map[string]bool{}
		for _, sourceKey := range account.SourceKeys {
			if seenSources[sourceKey] {
				return fmt.Errorf("editorial memory account contains duplicate sources")
			}
			if _, ok := catalog.Entry(sourceKey); !ok {
				return fmt.Errorf("editorial memory account contains an unavailable source")
			}
			seenSources[sourceKey] = true
		}
	}
	encoded, err := json.Marshal(memory)
	if err != nil || len(encoded) > MaxEditorialMemoryBytes {
		return fmt.Errorf("editorial memory exceeds the byte ceiling")
	}
	return nil
}

func ValidateEditorialMemoryArtifact(artifact EditorialMemoryArtifact, catalog SourceCatalog) error {
	memory := EditorialMemory{
		SchemaVersion: artifact.SchemaVersion,
		Language:      artifact.Language,
		Accounts:      artifact.Accounts,
	}
	if err := ValidateEditorialMemory(memory, catalog); err != nil {
		return err
	}
	if len(artifact.Anchors) == 0 || len(artifact.Anchors) > len(memory.Accounts)*MaxEditorialAnchorsPerAccount {
		return fmt.Errorf("editorial memory artifact requires bounded source anchors")
	}
	anchoredSourcesByAccount := make(map[string]map[string]bool, len(memory.Accounts))
	seenAnchors := map[string]bool{}
	for _, anchor := range artifact.Anchors {
		account, ok := memory.Account(anchor.AccountKey)
		entry, sourceAvailable := catalog.Entry(anchor.SourceKey)
		excerptBytes := []byte(anchor.Excerpt)
		anchorKey := fmt.Sprintf("%s:%s:%d:%d:%s", anchor.AccountKey, anchor.SourceKey, anchor.Offset, anchor.ByteSize, anchor.SHA256)
		if !ok || !sourceAvailable || !containsEditorialSource(account.SourceKeys, anchor.SourceKey) ||
			strings.TrimSpace(anchor.Excerpt) == "" || !utf8.Valid(excerptBytes) || anchor.Offset < 0 ||
			anchor.ByteSize != len(excerptBytes) || anchor.ByteSize > MaxSourceQuoteBytes ||
			anchor.Offset > entry.ReadableBytes-anchor.ByteSize || !validSourceAccessSHA256(anchor.SHA256) ||
			anchor.SHA256 != sourceAccessSHA256(excerptBytes) || seenAnchors[anchorKey] {
			return fmt.Errorf("editorial memory source anchor is invalid")
		}
		seenAnchors[anchorKey] = true
		if anchoredSourcesByAccount[anchor.AccountKey] == nil {
			anchoredSourcesByAccount[anchor.AccountKey] = map[string]bool{}
		}
		anchoredSourcesByAccount[anchor.AccountKey][anchor.SourceKey] = true
	}
	for _, account := range memory.Accounts {
		for _, sourceKey := range account.SourceKeys {
			if !anchoredSourcesByAccount[account.AccountKey][sourceKey] {
				return fmt.Errorf("editorial memory account source lacks an exact anchor")
			}
		}
	}
	encoded, err := json.Marshal(artifact)
	if err != nil || len(encoded) > MaxEditorialMemoryBytes {
		return fmt.Errorf("editorial memory exceeds the byte ceiling")
	}
	return nil
}

func containsEditorialSource(sourceKeys []string, want string) bool {
	for _, sourceKey := range sourceKeys {
		if sourceKey == want {
			return true
		}
	}
	return false
}

func EditorialMemoryFromArtifact(artifact EditorialMemoryArtifact) EditorialMemory {
	return EditorialMemory{
		SchemaVersion: artifact.SchemaVersion,
		Language:      artifact.Language,
		Accounts:      append([]EditorialAccount(nil), artifact.Accounts...),
	}
}

func (memory EditorialMemory) Account(accountKey string) (EditorialAccount, bool) {
	for _, account := range memory.Accounts {
		if account.AccountKey == accountKey {
			return account, true
		}
	}
	return EditorialAccount{}, false
}

func EditorialAccountSourceKeys(memory EditorialMemory, accountKeys []string) ([]string, error) {
	if len(accountKeys) == 0 {
		return nil, fmt.Errorf("editorial account binding is required")
	}
	seenAccounts := map[string]bool{}
	seenSources := map[string]bool{}
	sourceKeys := make([]string, 0)
	for _, accountKey := range accountKeys {
		if seenAccounts[accountKey] {
			return nil, fmt.Errorf("editorial account binding contains duplicates")
		}
		seenAccounts[accountKey] = true
		account, ok := memory.Account(accountKey)
		if !ok {
			return nil, fmt.Errorf("editorial account binding is unavailable")
		}
		for _, sourceKey := range account.SourceKeys {
			if seenSources[sourceKey] {
				continue
			}
			seenSources[sourceKey] = true
			sourceKeys = append(sourceKeys, sourceKey)
		}
	}
	return sourceKeys, nil
}
