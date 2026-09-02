package reportilphase0

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime"
	"net/netip"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/agentcapability"
	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/pdfdocument"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

const (
	MaxSourceBytes = 256 * 1024
	MaxPacketBytes = 512 * 1024
)

type SourcePacket struct {
	SchemaVersion string         `json:"schema_version"`
	MissionID     string         `json:"mission_id"`
	Sources       []PacketSource `json:"sources"`
	SHA256        string         `json:"sha256"`
}

type PacketSource struct {
	SnapshotReceipt    string            `json:"snapshot_receipt"`
	ObservationReceipt string            `json:"observation_receipt,omitempty"`
	Artifacts          []SourceArtifact  `json:"artifacts,omitempty"`
	Observation        map[string]string `json:"observation,omitempty"`
	Content            string            `json:"content"`
}

type SourceArtifact struct {
	ArtifactID string `json:"artifact_id"`
	SHA256     string `json:"sha256"`
	ByteSize   int64  `json:"byte_size"`
	MediaType  string `json:"media_type"`
}

type Provider interface {
	Run(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error)
}

type SourceReader interface {
	ListSourceSnapshots(context.Context, string) ([]reportilcontract.SourceSnapshot, error)
	GetArtifact(context.Context, string) (reportilcontract.Artifact, error)
	ReadLive(context.Context, string, string, int64) (reportilcontract.LocalRead, error)
}

func BuildSourcePacket(ctx context.Context, reader SourceReader, missionID string) (SourcePacket, error) {
	missionID = strings.TrimSpace(missionID)
	if missionID == "" {
		return SourcePacket{}, fmt.Errorf("source packet requires a mission")
	}
	snapshots, err := reader.ListSourceSnapshots(ctx, missionID)
	if err != nil {
		return SourcePacket{}, err
	}
	if len(snapshots) == 0 {
		return SourcePacket{}, fmt.Errorf("source packet requires at least one accepted source")
	}
	for _, snapshot := range snapshots {
		if snapshot.Active && strings.TrimSpace(snapshot.MissionID) != missionID {
			return SourcePacket{}, fmt.Errorf("source packet contains a cross-mission source")
		}
	}
	sort.Slice(snapshots, func(i, j int) bool { return snapshots[i].SnapshotID < snapshots[j].SnapshotID })
	packet := SourcePacket{SchemaVersion: "plasma.report_source_packet.experimental.v1", MissionID: missionID}
	for _, snapshot := range snapshots {
		if !snapshot.Active {
			continue
		}
		item, err := readPacketSource(ctx, reader, snapshot)
		if err != nil {
			return SourcePacket{}, err
		}
		packet.Sources = append(packet.Sources, item)
	}
	if len(packet.Sources) == 0 {
		return SourcePacket{}, fmt.Errorf("source packet has no active accepted sources")
	}
	preimage := struct {
		SchemaVersion string         `json:"schema_version"`
		MissionID     string         `json:"mission_id"`
		Sources       []PacketSource `json:"sources"`
	}{packet.SchemaVersion, packet.MissionID, packet.Sources}
	raw, err := json.Marshal(preimage)
	if err != nil {
		return SourcePacket{}, err
	}
	if len(raw) > MaxPacketBytes {
		return SourcePacket{}, fmt.Errorf("source packet exceeds size ceiling")
	}
	packet.SHA256 = sha256Hex(raw)
	if len(mustJSON(packet)) > MaxPacketBytes {
		return SourcePacket{}, fmt.Errorf("source packet exceeds size ceiling")
	}
	return packet, nil
}

func readPacketSource(ctx context.Context, reader SourceReader, snapshot reportilcontract.SourceSnapshot) (PacketSource, error) {
	item := PacketSource{SnapshotReceipt: snapshotReceipt(snapshot)}
	if snapshot.RetrievalPolicy == source.RetrievalPolicyLiveReference {
		read, err := reader.ReadLive(ctx, snapshot.MissionID, snapshot.SnapshotID, MaxSourceBytes)
		if err != nil {
			return PacketSource{}, err
		}
		if read.Binary || read.Truncated || strings.TrimSpace(read.Content) == "" {
			return PacketSource{}, fmt.Errorf("source packet rejects binary, empty, or truncated local source")
		}
		if err := validateProviderContent(read.Content); err != nil {
			return PacketSource{}, err
		}
		if read.SHA256 != "" && sha256Hex([]byte(read.Content)) != read.SHA256 {
			return PacketSource{}, fmt.Errorf("source packet rejects changed live-source continuity")
		}
		if read.ObservationReceipt == "" {
			return PacketSource{}, fmt.Errorf("source packet requires a live observation receipt")
		}
		item.Content = read.Content
		item.ObservationReceipt = read.ObservationReceipt
		item.Observation = map[string]string{"size": fmt.Sprintf("%d", read.Size)}
		if strings.TrimSpace(read.MTime) != "" {
			item.Observation["mtime"] = read.MTime
		}
		if strings.TrimSpace(read.SHA256) != "" {
			item.Observation["sha256"] = read.SHA256
		}
		if strings.TrimSpace(read.Extraction) != "" {
			item.Observation["extraction"] = read.Extraction
		}
		return item, nil
	}
	if len(snapshot.ArtifactIDs) == 0 {
		return PacketSource{}, fmt.Errorf("source packet rejects source without artifact")
	}
	for _, artifactID := range snapshot.ArtifactIDs {
		artifact, err := reader.GetArtifact(ctx, artifactID)
		if err != nil {
			return PacketSource{}, err
		}
		if artifact.MissionID != snapshot.MissionID || artifact.SHA256 == "" {
			return PacketSource{}, fmt.Errorf("source packet artifact hash or mission mismatch")
		}
		base, _, parseErr := mime.ParseMediaType(artifact.MediaType)
		if parseErr != nil {
			base = artifact.MediaType
		}
		var content []byte
		switch {
		case strings.HasPrefix(base, "text/"):
			if sha256Hex(artifact.Content) != artifact.SHA256 || !utf8.Valid(artifact.Content) {
				return PacketSource{}, fmt.Errorf("source packet rejects binary text artifact")
			}
			content = artifact.Content
		case pdfdocument.IsPDFMediaType(base) || pdfdocument.IsPDFBytes(artifact.Content):
			if sha256Hex(artifact.Content) != artifact.SHA256 {
				return PacketSource{}, fmt.Errorf("source packet artifact hash or mission mismatch")
			}
			extracted, extractErr := pdfdocument.Extract(artifact.Content)
			if extractErr != nil || extracted.Truncated {
				return PacketSource{}, fmt.Errorf("source packet PDF extraction failed or was truncated")
			}
			content = []byte(extracted.Text)
		default:
			return PacketSource{}, fmt.Errorf("source packet rejects unsupported artifact media type")
		}
		if err := validateProviderContent(string(content)); err != nil {
			return PacketSource{}, err
		}
		if len(content) == 0 || len(content) > MaxSourceBytes {
			return PacketSource{}, fmt.Errorf("source packet rejects empty or oversized source")
		}
		if item.Content != "" {
			item.Content += "\n\n"
		}
		item.Content += string(content)
		item.Artifacts = append(item.Artifacts, SourceArtifact{ArtifactID: artifact.ArtifactID, SHA256: artifact.SHA256, ByteSize: artifact.ByteSize, MediaType: artifact.MediaType})
	}
	if strings.TrimSpace(item.Content) == "" {
		return PacketSource{}, fmt.Errorf("source packet rejects empty source content")
	}
	if snapshot.ContentHash != "" && !strings.EqualFold(snapshot.ContentHash, snapshotHashValue(item.Artifacts)) {
		return PacketSource{}, fmt.Errorf("source packet snapshot hash mismatch")
	}
	return item, nil
}

func snapshotReceipt(snapshot reportilcontract.SourceSnapshot) string {
	return reportilcontract.SourceSnapshotReceipt(snapshot.SnapshotID, snapshot.ContentHash)
}
func snapshotHashValue(artifacts []SourceArtifact) string {
	if len(artifacts) == 1 {
		return artifacts[0].SHA256
	}
	var values []byte
	for _, item := range artifacts {
		values = append(values, item.ArtifactID...)
		values = append(values, 0)
		values = append(values, item.SHA256...)
		values = append(values, 0)
	}
	return sha256Hex(values)
}

var credentialPattern = regexp.MustCompile(`(?i)(authorization\s*[:=]|bearer\s+[A-Za-z0-9._~+/=-]{8,}|api[_-]?key\s*[:=]|access[_-]?token\s*[:=]|refresh[_-]?token\s*[:=]|client[_-]?secret\s*[:=]|cookie\s*[:=]|session[_-]?token\s*[:=])`)

func validateProviderContent(content string) error {
	if credentialPattern.MatchString(content) {
		return fmt.Errorf("source packet contains restricted credential material")
	}
	for _, value := range strings.Fields(content) {
		u, err := url.Parse(strings.Trim(value, `<>\"'(),;`))
		if err != nil || u.Scheme == "" {
			continue
		}
		if strings.EqualFold(u.Scheme, "file") || u.User != nil {
			return fmt.Errorf("source packet contains restricted URL material")
		}
		host := strings.ToLower(u.Hostname())
		if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".ts.net") {
			return fmt.Errorf("source packet contains restricted URL material")
		}
		if ip, err := netip.ParseAddr(host); err == nil {
			ip = ip.Unmap()
			if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || netip.MustParsePrefix("100.64.0.0/10").Contains(ip) {
				return fmt.Errorf("source packet contains restricted URL material")
			}
		}
	}
	return nil
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
func reportILProfile() agentcapability.Profile { return agentcapability.ReportIL() }
func mustJSON(value any) []byte                { raw, _ := json.Marshal(value); return raw }
