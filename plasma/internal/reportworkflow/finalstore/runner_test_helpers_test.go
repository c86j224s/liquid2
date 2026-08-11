package finalstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

type fakeService struct {
	calls       []string
	created     []artifact.CreateRequest
	appended    []ledger.AppendRequest
	atomicCalls int
	createErr   error
	appendErr   error
	atomicErr   error
}

func (fake *fakeService) AppendEvent(_ context.Context, req ledger.AppendRequest) (ledger.Event, error) {
	fake.calls = append(fake.calls, "append")
	fake.appended = append(fake.appended, req)
	if fake.appendErr != nil {
		return ledger.Event{}, fake.appendErr
	}
	return ledger.Event{EventID: req.EventID, MissionID: req.MissionID, EventType: req.EventType, Producer: req.Producer, Payload: req.Payload, CreatedAt: time.Now()}, nil
}

func (fake *fakeService) CreateRawArtifact(_ context.Context, req artifact.CreateRequest) (artifact.Raw, error) {
	fake.calls = append(fake.calls, "create")
	fake.created = append(fake.created, req)
	if fake.createErr != nil {
		return artifact.Raw{}, fake.createErr
	}
	return rawArtifact(req), nil
}

func (fake *fakeService) CreateRawArtifactWithEvent(_ context.Context, req artifact.CreateRequest, build func(artifact.Raw) ledger.AppendRequest) (artifact.Raw, ledger.Event, error) {
	fake.calls = append(fake.calls, "atomic")
	fake.atomicCalls++
	raw := rawArtifact(req)
	eventReq := build(raw)
	if fake.atomicErr != nil {
		return artifact.Raw{}, ledger.Event{}, fake.atomicErr
	}
	return raw, ledger.Event{EventID: eventReq.EventID, MissionID: eventReq.MissionID, EventType: eventReq.EventType, Producer: eventReq.Producer, Payload: eventReq.Payload, CreatedAt: time.Now()}, nil
}

func rawArtifact(req artifact.CreateRequest) artifact.Raw {
	return artifact.Raw{
		ArtifactID: req.ArtifactID, MissionID: req.MissionID, MediaType: req.MediaType,
		ByteSize: int64(len(req.Content)), SHA256: sha(string(req.Content)),
		Filename: req.Filename, Producer: req.Producer, Content: append([]byte(nil), req.Content...),
	}
}

type fakeGateReader struct {
	record GateRecord
	err    error
}

func (fake *fakeGateReader) ReadFinalGate(context.Context, GateReadRequest) (GateRecord, error) {
	if fake.err != nil {
		return GateRecord{}, fake.err
	}
	return fake.record, nil
}

type idRecorder struct {
	order  []string
	counts map[string]int
}

func (rec *idRecorder) next(prefix string) string {
	if rec.counts == nil {
		rec.counts = map[string]int{}
	}
	rec.order = append(rec.order, prefix)
	rec.counts[prefix]++
	return fmt.Sprintf("%s_%d", prefix, rec.counts[prefix])
}

func eventPayload(t *testing.T, event ledger.Event) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func mustJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func sha(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
