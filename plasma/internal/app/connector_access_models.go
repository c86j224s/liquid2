package app

import "github.com/c86j224s/liquid2/plasma/internal/confluenceaccess"

const (
	// ConnectorAccessEvent* 값은 미션이 connector 접근을 허용/변경/해제한 장부 event다.
	ConnectorAccessEventEnabled  = confluenceaccess.AccessEventEnabled
	ConnectorAccessEventUpdated  = confluenceaccess.AccessEventUpdated
	ConnectorAccessEventDisabled = confluenceaccess.AccessEventDisabled

	// ConnectorAccessStatus* 값은 mission connector access projection의 상태다.
	ConnectorAccessStatusDisabled = confluenceaccess.AccessStatusDisabled
	ConnectorAccessStatusEnabled  = confluenceaccess.AccessStatusEnabled
	ConnectorAccessStatusInvalid  = confluenceaccess.AccessStatusInvalid
)

// ConnectorAccessProjection은 한 미션이 connector connection을 사용할 수 있는지에
// 대한 현재 view다.
type ConnectorAccessProjection = confluenceaccess.AccessProjection

// SetConnectorAccessRequest는 미션별 connector 접근 권한을 변경하는 입력이다.
type SetConnectorAccessRequest = confluenceaccess.SetAccessRequest

// ConnectorAccessChangeResult는 connector 접근 권한 변경 결과와 기록 event다.
type ConnectorAccessChangeResult = confluenceaccess.AccessChangeResult
