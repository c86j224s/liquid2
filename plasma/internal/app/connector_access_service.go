package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type connectorAccessPayload struct {
	ConnectorID  string `json:"connector_id"`
	ConnectionID string `json:"connection_id,omitempty"`
	CloudID      string `json:"cloud_id,omitempty"`
	SpaceKey     string `json:"space_key,omitempty"`
}

func (s *Service) GetMissionConnectorAccess(ctx context.Context, missionID string, connectorID string) (ConnectorAccessProjection, error) {
	if err := validateID("mis_", strings.TrimSpace(missionID)); err != nil {
		return ConnectorAccessProjection{}, err
	}
	connectorID, err := normalizeConnectorAccessID(connectorID)
	if err != nil {
		return ConnectorAccessProjection{}, err
	}
	events, err := s.store.ListLedgerEvents(ctx, strings.TrimSpace(missionID))
	if err != nil {
		return ConnectorAccessProjection{}, err
	}
	return s.projectConnectorAccess(ctx, strings.TrimSpace(missionID), connectorID, events), nil
}

func (s *Service) SetMissionConnectorAccess(ctx context.Context, req SetConnectorAccessRequest) (ConnectorAccessChangeResult, error) {
	req.MissionID = strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", req.MissionID); err != nil {
		return ConnectorAccessChangeResult{}, err
	}
	connectorID, err := normalizeConnectorAccessID(req.ConnectorID)
	if err != nil {
		return ConnectorAccessChangeResult{}, err
	}
	req.ConnectorID = connectorID
	if err := validateID("evt_", strings.TrimSpace(req.EventID)); err != nil {
		return ConnectorAccessChangeResult{}, err
	}
	if !connectorAccessUserProducer(req.Producer) {
		return ConnectorAccessChangeResult{}, fmt.Errorf("%w: connector access must be changed by a user action", ErrInvalidInput)
	}
	if req.Enabled {
		if err := s.validateConfluenceConnectorAccessTarget(ctx, req.ConnectionID, req.CloudID); err != nil {
			return ConnectorAccessChangeResult{}, err
		}
	}
	events, err := s.store.ListLedgerEvents(ctx, req.MissionID)
	if err != nil {
		return ConnectorAccessChangeResult{}, err
	}
	current := s.projectConnectorAccess(ctx, req.MissionID, connectorID, events)
	eventType := ConnectorAccessEventDisabled
	if req.Enabled {
		eventType = ConnectorAccessEventEnabled
		if current.Enabled {
			eventType = ConnectorAccessEventUpdated
		}
	}
	payload := connectorAccessPayload{
		ConnectorID:  connectorID,
		ConnectionID: strings.TrimSpace(req.ConnectionID),
		CloudID:      strings.TrimSpace(req.CloudID),
		SpaceKey:     strings.TrimSpace(req.SpaceKey),
	}
	if !req.Enabled {
		payload.ConnectionID = ""
		payload.CloudID = ""
		payload.SpaceKey = ""
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return ConnectorAccessChangeResult{}, err
	}
	event, err := s.AppendEvent(ctx, AppendEventRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: req.MissionID,
		EventType: eventType,
		Producer:  Producer{Type: strings.TrimSpace(req.Producer.Type), ID: strings.TrimSpace(req.Producer.ID)},
		Payload:   encoded,
	})
	if err != nil {
		return ConnectorAccessChangeResult{}, err
	}
	events = append(events, event)
	return ConnectorAccessChangeResult{
		Access: s.projectConnectorAccess(ctx, req.MissionID, connectorID, events),
		Event:  event,
	}, nil
}

func (s *Service) projectConnectorAccess(ctx context.Context, missionID string, connectorID string, events []LedgerEvent) ConnectorAccessProjection {
	projection := ConnectorAccessProjection{
		MissionID:   missionID,
		ConnectorID: connectorID,
		Status:      ConnectorAccessStatusDisabled,
	}
	for _, event := range events {
		switch event.EventType {
		case ConnectorAccessEventEnabled, ConnectorAccessEventUpdated, ConnectorAccessEventDisabled:
		default:
			continue
		}
		var payload connectorAccessPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.ConnectorID) != connectorID {
			continue
		}
		projection.LastEventID = event.EventID
		projection.LastSequence = event.Sequence
		if event.EventType == ConnectorAccessEventDisabled {
			projection.Enabled = false
			projection.ConnectionID = ""
			projection.CloudID = ""
			projection.SpaceKey = ""
			projection.Status = ConnectorAccessStatusDisabled
			projection.InvalidReason = ""
			continue
		}
		projection.Enabled = true
		projection.ConnectionID = strings.TrimSpace(payload.ConnectionID)
		projection.CloudID = strings.TrimSpace(payload.CloudID)
		projection.SpaceKey = strings.TrimSpace(payload.SpaceKey)
		projection.Status = ConnectorAccessStatusEnabled
		projection.InvalidReason = ""
	}
	if projection.Enabled {
		if reason := s.confluenceConnectorAccessInvalidReason(ctx, projection.ConnectionID, projection.CloudID); reason != "" {
			projection.Status = ConnectorAccessStatusInvalid
			projection.InvalidReason = reason
		}
	}
	return projection
}

func (s *Service) validateConfluenceConnectorAccessTarget(ctx context.Context, connectionID string, cloudID string) error {
	connectionID = strings.TrimSpace(connectionID)
	cloudID = strings.TrimSpace(cloudID)
	if err := validateID("cnf_", connectionID); err != nil {
		return err
	}
	if cloudID == "" {
		return fmt.Errorf("%w: confluence cloud_id is required", ErrInvalidInput)
	}
	connection, err := s.GetConfluenceConnection(ctx, connectionID)
	if err != nil {
		return err
	}
	if connection.Revoked {
		return fmt.Errorf("%w: confluence connection is revoked", ErrInvalidInput)
	}
	for _, site := range connection.Sites {
		if strings.TrimSpace(site.CloudID) == cloudID {
			return nil
		}
	}
	return fmt.Errorf("%w: confluence cloud_id does not belong to the selected connection", ErrInvalidInput)
}

func (s *Service) confluenceConnectorAccessInvalidReason(ctx context.Context, connectionID string, cloudID string) string {
	connection, err := s.GetConfluenceConnection(ctx, connectionID)
	if err != nil {
		return "connection_unavailable"
	}
	if connection.Revoked {
		return "connection_revoked"
	}
	for _, site := range connection.Sites {
		if strings.TrimSpace(site.CloudID) == strings.TrimSpace(cloudID) {
			return ""
		}
	}
	return "cloud_unavailable"
}

func normalizeConnectorAccessID(connectorID string) (string, error) {
	connectorID = strings.TrimSpace(connectorID)
	if connectorID != ConfluenceConnectorID {
		return "", fmt.Errorf("%w: unsupported connector access id", ErrInvalidInput)
	}
	return connectorID, nil
}

func connectorAccessUserProducer(producer Producer) bool {
	return strings.TrimSpace(producer.Type) == "user" && strings.TrimSpace(producer.ID) != ""
}
