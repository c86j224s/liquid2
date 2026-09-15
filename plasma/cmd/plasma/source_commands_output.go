package main

import uploadsource "github.com/c86j224s/liquid2/plasma/internal/source"

import (
	"errors"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"io"
	"time"
)

type cliRawArtifactAPIResponse struct {
	ArtifactID string
	MissionID  string
	MediaType  string
	ByteSize   int64
	SHA256     string
	StorageURI string
	Filename   string
	Producer   ledger.Producer
	CreatedAt  time.Time
}

func cliRawArtifactMetadata(artifact artifactcontract.Raw) map[string]any {
	return uploadsource.UploadedArtifactMetadata(artifact)
}

func cliRawArtifactResponse(artifact artifactcontract.Raw) cliRawArtifactAPIResponse {
	return cliRawArtifactAPIResponse{
		ArtifactID: artifact.ArtifactID,
		MissionID:  artifact.MissionID,
		MediaType:  artifact.MediaType,
		ByteSize:   artifact.ByteSize,
		SHA256:     artifact.SHA256,
		StorageURI: artifact.StorageURI,
		Filename:   artifact.Filename,
		Producer:   artifact.Producer,
		CreatedAt:  artifact.CreatedAt,
	}
}

func cliLedgerEventID(event *ledger.Event) string {
	if event == nil {
		return ""
	}
	return event.EventID
}

func writeSourceCommandError(stderr io.Writer, command string, err error) {
	fmt.Fprintf(stderr, "%s: %v\n", command, err)
}

func cliErrorCode(err error) int {
	if confluenceErr, ok := confluencesource.ConfluenceErrorDetails(err); ok {
		if confluenceErr.HTTPStatus >= 400 && confluenceErr.HTTPStatus < 500 {
			return 2
		}
	}
	if errors.Is(err, app.ErrInvalidInput) {
		return 2
	}
	return 1
}
