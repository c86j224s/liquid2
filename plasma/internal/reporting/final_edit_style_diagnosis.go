package reporting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	FinalEditStyleOperationDiagnosesField        = "style_operation_diagnoses"
	FinalEditStyleOperationDiagnosesVersionField = "style_operation_diagnoses_version"
	FinalEditStyleOperationDiagnosesVersion      = 1
)

type FinalEditStyleOperationDiagnosis struct {
	OperationOrdinal int    `json:"operation_ordinal"`
	Category         string `json:"category"`
	Reason           string `json:"reason"`
	MatchText        string `json:"match_text"`
	Replacement      string `json:"replacement"`
	Occurrence       int    `json:"occurrence"`
}

type legacyFinalEditStyleOperationDiagnosis struct {
	OperationOrdinal int    `json:"operation_ordinal"`
	Category         string `json:"category"`
}

var finalEditStyleDiagnosisCategories = map[string]bool{
	"opaque_or_strained_mapping":  true,
	"unnatural_collocation":       true,
	"vague_reference":             true,
	"nominalized_or_bureaucratic": true,
	"compressed_abstraction":      true,
	"report_process_meta":         true,
	"formulaic_transition":        true,
}

func ValidateFinalEditStyleDiagnosisCategory(category string) error {
	if category != strings.TrimSpace(category) || !finalEditStyleDiagnosisCategories[category] {
		return fmt.Errorf("%w: style operation diagnosis category is invalid", app.ErrInvalidInput)
	}
	return nil
}

func ValidateFinalEditStyleOperationDiagnoses(operationCount int, diagnoses []FinalEditStyleOperationDiagnosis, detailed bool) error {
	if operationCount < 0 {
		return fmt.Errorf("%w: style operation count is invalid", app.ErrInvalidInput)
	}
	if operationCount != len(diagnoses) {
		return fmt.Errorf("%w: style operation diagnoses count differs from operation count", app.ErrInvalidInput)
	}
	for index, diagnosis := range diagnoses {
		if diagnosis.OperationOrdinal != index+1 || diagnosis.OperationOrdinal <= 0 {
			return fmt.Errorf("%w: style operation diagnosis ordinal is invalid", app.ErrInvalidInput)
		}
		if err := ValidateFinalEditStyleDiagnosisCategory(diagnosis.Category); err != nil {
			return err
		}
		if detailed {
			if strings.TrimSpace(diagnosis.Reason) == "" || diagnosis.Reason != strings.TrimSpace(diagnosis.Reason) {
				return fmt.Errorf("%w: style operation diagnosis reason is invalid", app.ErrInvalidInput)
			}
			if diagnosis.MatchText == "" {
				return fmt.Errorf("%w: style operation diagnosis match text is invalid", app.ErrInvalidInput)
			}
			if diagnosis.Occurrence <= 0 {
				return fmt.Errorf("%w: style operation diagnosis occurrence is invalid", app.ErrInvalidInput)
			}
		}
	}
	return nil
}

func decodeFinalEditStyleOperationDiagnosesPayload(value any, operationCount int, detailed bool) ([]FinalEditStyleOperationDiagnosis, bool, error) {
	if value == nil {
		return nil, true, fmt.Errorf("%w: style operation diagnoses payload is invalid", app.ErrConflict)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, true, fmt.Errorf("%w: style operation diagnoses payload is invalid", app.ErrConflict)
	}
	if detailed && !finalEditStyleDiagnosisRecordsHaveExactFields(raw, []string{"operation_ordinal", "category", "reason", "match_text", "replacement", "occurrence"}) {
		return nil, true, fmt.Errorf("%w: style operation diagnoses payload is invalid", app.ErrConflict)
	}
	var records []FinalEditStyleOperationDiagnosis
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&records); err != nil {
		return nil, true, fmt.Errorf("%w: style operation diagnoses payload is invalid", app.ErrConflict)
	}
	if decoder.More() {
		return nil, true, fmt.Errorf("%w: style operation diagnoses payload is invalid", app.ErrConflict)
	}
	if err := ValidateFinalEditStyleOperationDiagnoses(operationCount, records, detailed); err != nil {
		return nil, true, fmt.Errorf("%w: style operation diagnoses payload is invalid", app.ErrConflict)
	}
	return append([]FinalEditStyleOperationDiagnosis(nil), records...), true, nil
}

func decodeLegacyFinalEditStyleOperationDiagnosesPayload(value any, operationCount int) ([]FinalEditStyleOperationDiagnosis, bool, error) {
	if value == nil {
		return nil, true, fmt.Errorf("%w: style operation diagnoses payload is invalid", app.ErrConflict)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, true, fmt.Errorf("%w: style operation diagnoses payload is invalid", app.ErrConflict)
	}
	if !finalEditStyleDiagnosisRecordsHaveExactFields(raw, []string{"operation_ordinal", "category"}) {
		return nil, true, fmt.Errorf("%w: style operation diagnoses payload is invalid", app.ErrConflict)
	}
	var legacyRecords []legacyFinalEditStyleOperationDiagnosis
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&legacyRecords); err != nil {
		return nil, true, fmt.Errorf("%w: style operation diagnoses payload is invalid", app.ErrConflict)
	}
	if decoder.More() {
		return nil, true, fmt.Errorf("%w: style operation diagnoses payload is invalid", app.ErrConflict)
	}
	records := make([]FinalEditStyleOperationDiagnosis, 0, len(legacyRecords))
	for _, record := range legacyRecords {
		records = append(records, FinalEditStyleOperationDiagnosis{
			OperationOrdinal: record.OperationOrdinal,
			Category:         record.Category,
		})
	}
	if err := ValidateFinalEditStyleOperationDiagnoses(operationCount, records, false); err != nil {
		return nil, true, fmt.Errorf("%w: style operation diagnoses payload is invalid", app.ErrConflict)
	}
	return records, true, nil
}

func finalEditStyleDiagnosisRecordsHaveExactFields(raw []byte, fields []string) bool {
	var records []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &records); err != nil {
		return false
	}
	for _, record := range records {
		if len(record) != len(fields) {
			return false
		}
		for _, field := range fields {
			if _, ok := record[field]; !ok {
				return false
			}
		}
	}
	return true
}

func equalFinalEditStyleOperationDiagnoses(left, right []FinalEditStyleOperationDiagnosis) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func equalFinalEditStyleOperationDiagnosesForReplay(stored, retry []FinalEditStyleOperationDiagnosis) bool {
	if finalEditStyleOperationDiagnosesAreLegacy(stored) {
		return equalLegacyFinalEditStyleOperationDiagnoses(stored, retry)
	}
	return equalFinalEditStyleOperationDiagnoses(stored, retry)
}

func finalEditStyleOperationDiagnosesAreLegacy(records []FinalEditStyleOperationDiagnosis) bool {
	for _, record := range records {
		if record.Reason != "" || record.MatchText != "" || record.Replacement != "" || record.Occurrence != 0 {
			return false
		}
	}
	return true
}

func equalLegacyFinalEditStyleOperationDiagnoses(left, right []FinalEditStyleOperationDiagnosis) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].OperationOrdinal != right[index].OperationOrdinal || left[index].Category != right[index].Category {
			return false
		}
	}
	return true
}
