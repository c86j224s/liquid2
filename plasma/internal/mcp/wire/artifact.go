package wire

type RawArtifactOutput struct {
	ArtifactID string `json:"artifact_id"`
	MissionID  string `json:"mission_id"`
	MediaType  string `json:"media_type"`
	ByteSize   int64  `json:"byte_size"`
	SHA256     string `json:"sha256"`
	StorageURI string `json:"storage_uri"`
	Filename   string `json:"filename,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	ReadKind   string `json:"read_kind,omitempty"`
}
