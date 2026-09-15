package app

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
)

func confluenceSnapshotProducer(producer ledger.Producer) (ledger.Producer, error) {
	producer.Type = strings.TrimSpace(producer.Type)
	producer.ID = strings.TrimSpace(producer.ID)
	if producer.Type == "" && producer.ID == "" {
		return ledger.Producer{Type: "connector", ID: confluencesource.ConfluenceConnectorID}, nil
	}
	if producer.Type != "connector" || producer.ID != confluencesource.ConfluenceConnectorID {
		return ledger.Producer{}, fmt.Errorf("%w: confluence snapshot producer must be connector/confluence", ErrInvalidInput)
	}
	return producer, nil
}

func confluenceSnapshotFilename(externalSourceID string) string {
	externalSourceID = strings.TrimSpace(externalSourceID)
	if externalSourceID == "" {
		return "plasma-confluence-snapshot.json"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_")
	return "plasma-confluence-snapshot-" + replacer.Replace(externalSourceID) + ".json"
}
