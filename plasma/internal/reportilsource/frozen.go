package reportilsource

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/source"
	"strings"
)

// FrozenSourceReaders supplies ordered reads; binding, budgets and trace stay in MCP.
type FrozenSourceReaders struct {
	GetSourceSnapshot func(context.Context, string) (source.Snapshot, error)
	GetRawArtifact    func(context.Context, string) (artifact.Raw, error)
	ReadLiveSource    func(context.Context, string, string, int) (LiveSourceRead, error)
}
type LiveSourceRead struct {
	Content, Extraction string
	Binary, Truncated   bool
}

// ResolveFrozenSource checks frozen identities and canonical content. The repeated
// single-artifact read deliberately preserves the historical access sequence.
func ResolveFrozenSource(ctx context.Context, readers FrozenSourceReaders, binding reportilcontract.SourceAccessBinding, entry reportilcontract.SourceCatalogEntry) (Readable, error) {
	snapshot, err := readers.GetSourceSnapshot(ctx, entry.SnapshotID)
	if err != nil {
		return Readable{}, err
	}
	if snapshot.MissionID != binding.Catalog.MissionID || (snapshot.State.Removed || strings.TrimSpace(snapshot.State.State) == source.StateRemoved) || snapshot.State.Superseded {
		return Readable{}, fmt.Errorf("frozen report IL source is no longer active")
	}
	retrievalPolicy := strings.TrimSpace(snapshot.Access.RetrievalPolicy)
	if retrievalPolicy == "" {
		retrievalPolicy = source.RetrievalPolicySnapshotOnly
	}
	if retrievalPolicy != entry.RetrievalPolicy {
		return Readable{}, fmt.Errorf("frozen report IL source retrieval policy changed")
	}
	if reportilcontract.SourceSnapshotReceipt(snapshot.SnapshotID, snapshot.ContentHash.Value) != entry.SnapshotReceipt || !strings.EqualFold(snapshot.ContentHash.Value, entry.ContentHash) {
		return Readable{}, fmt.Errorf("frozen report IL source identity changed")
	}
	var readable Readable
	if entry.RetrievalPolicy == source.RetrievalPolicyLiveReference {
		result, err := readers.ReadLiveSource(ctx, binding.Catalog.MissionID, entry.SnapshotID, MaxReadableBytes)
		if err != nil {
			return Readable{}, err
		}
		if result.Binary || result.Truncated {
			return Readable{}, fmt.Errorf("frozen live report IL source cannot be read as complete text")
		}
		readable, err = Extract([]byte(result.Content), "text/plain")
		if err != nil {
			return Readable{}, err
		}
		if strings.TrimSpace(result.Extraction) != "" {
			readable.Extraction = result.Extraction
		} else {
			readable.Extraction = "live_text"
		}
	} else {
		var combined strings.Builder
		if len(entry.Artifacts) == 0 || len(entry.Artifacts) != len(snapshot.ArtifactIDs) {
			return Readable{}, fmt.Errorf("frozen report IL artifact set changed")
		}
		for index, receipt := range entry.Artifacts {
			if snapshot.ArtifactIDs[index] != receipt.ArtifactID {
				return Readable{}, fmt.Errorf("frozen report IL artifact order changed")
			}
			artifact, err := readers.GetRawArtifact(ctx, receipt.ArtifactID)
			if err != nil {
				return Readable{}, err
			}
			if artifact.MissionID != binding.Catalog.MissionID || !strings.EqualFold(artifact.SHA256, receipt.SHA256) || !strings.EqualFold(sha256Hex(artifact.Content), receipt.SHA256) || artifact.ByteSize != receipt.ByteSize || int64(len(artifact.Content)) != receipt.ByteSize || artifact.MediaType != receipt.MediaType {
				return Readable{}, fmt.Errorf("frozen report IL artifact receipt changed")
			}
			part, err := Extract(artifact.Content, artifact.MediaType)
			if err != nil {
				return Readable{}, err
			}
			if combined.Len() > 0 {
				combined.WriteString("\n\n")
			}
			combined.WriteString(part.Text)
			if len(entry.Artifacts) == 1 {
				readable.Extraction = part.Extraction
			}
		}
		readable, err = Extract([]byte(combined.String()), "text/plain")
		if err != nil {
			return Readable{}, err
		}
		if len(entry.Artifacts) == 1 {
			part, partErr := readers.GetRawArtifact(ctx, entry.Artifacts[0].ArtifactID)
			if partErr != nil {
				return Readable{}, partErr
			}
			partReadable, partErr := Extract(part.Content, part.MediaType)
			if partErr != nil {
				return Readable{}, partErr
			}
			readable.Extraction = partReadable.Extraction
		} else {
			readable.Extraction = "joined_artifact_text"
		}
	}
	if readable.SHA256 != entry.ReadableSHA256 || readable.ByteSize != entry.ReadableBytes || readable.Extraction != entry.Extraction {
		return Readable{}, fmt.Errorf("frozen report IL readable content changed")
	}
	return readable, nil
}
