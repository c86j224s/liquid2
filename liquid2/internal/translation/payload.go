package translation

import (
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/internal/app"
)

type TranslateDocumentPayload struct {
	DocumentID      string `json:"documentId"`
	SourceContentID string `json:"sourceContentId"`
	TargetLanguage  string `json:"targetLanguage"`
}

func EncodeTranslateDocumentPayload(documentID, sourceContentID, targetLanguage string) (string, error) {
	language, err := normalizeTargetLanguage(targetLanguage)
	if err != nil {
		return "", err
	}
	payload := TranslateDocumentPayload{
		DocumentID:      strings.TrimSpace(documentID),
		SourceContentID: strings.TrimSpace(sourceContentID),
		TargetLanguage:  language,
	}
	if err := payload.validate(); err != nil {
		return "", err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", invalidPayload("encode translate document payload", err)
	}
	return string(data), nil
}

func DecodeTranslateDocumentPayload(raw string) (TranslateDocumentPayload, error) {
	var payload TranslateDocumentPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return TranslateDocumentPayload{}, invalidPayload("decode translate document payload", err)
	}
	payload.DocumentID = strings.TrimSpace(payload.DocumentID)
	payload.SourceContentID = strings.TrimSpace(payload.SourceContentID)
	targetLanguage, err := normalizeTargetLanguage(payload.TargetLanguage)
	if err != nil {
		return TranslateDocumentPayload{}, err
	}
	payload.TargetLanguage = targetLanguage
	if err := payload.validate(); err != nil {
		return TranslateDocumentPayload{}, err
	}
	return payload, nil
}

func (payload TranslateDocumentPayload) validate() error {
	switch {
	case payload.DocumentID == "":
		return invalidPayload("document id is required")
	case payload.SourceContentID == "":
		return invalidPayload("source content id is required")
	case payload.TargetLanguage == "":
		return invalidPayload("target language is required")
	default:
		return nil
	}
}

func normalizeTargetLanguage(value string) (string, error) {
	language, err := app.NormalizeContentLanguage(value)
	if err != nil {
		return "", invalidPayload("target language is invalid", err)
	}
	return language, nil
}
