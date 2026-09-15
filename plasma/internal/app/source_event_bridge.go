package app

import (
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/sourceevents"
)

func sourceEventConnectorRef(connector sourcecontract.ConnectorRef) sourceevents.ConnectorRef {
	return sourceevents.ConnectorRef{
		ConnectorID:      connector.ConnectorID,
		ConnectorType:    connector.ConnectorType,
		ExternalSourceID: connector.ExternalSourceID,
		ExternalURI:      connector.ExternalURI,
		ExternalVersion:  connector.ExternalVersion,
		ConnectorVersion: connector.ConnectorVersion,
	}
}
