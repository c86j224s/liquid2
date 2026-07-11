package app

import (
	"errors"
	"fmt"
	"strings"
)

const (
	ConfluenceErrorCategoryAuth        = "confluence_auth"
	ConfluenceErrorCategoryPermission  = "confluence_permission"
	ConfluenceErrorCategoryNotFound    = "confluence_not_found"
	ConfluenceErrorCategoryRateLimited = "confluence_rate_limited"
	ConfluenceErrorCategoryValidation  = "confluence_validation"
	ConfluenceErrorCategoryConflict    = "confluence_conflict"
	ConfluenceErrorCategoryUpstream    = "confluence_upstream"

	ConfluenceErrorCodeUnauthorized  = "confluence_unauthorized"
	ConfluenceErrorCodeForbidden     = "confluence_forbidden"
	ConfluenceErrorCodeNotFound      = "confluence_not_found"
	ConfluenceErrorCodeRateLimited   = "confluence_rate_limited"
	ConfluenceErrorCodeVersionDrift  = "confluence_version_changed"
	ConfluenceErrorCodeCloudMismatch = "confluence_cloud_mismatch"
	ConfluenceErrorCodePageMismatch  = "confluence_page_mismatch"
	ConfluenceErrorCodeTooLarge      = "confluence_page_too_large"
	ConfluenceErrorCodeTokenExpired  = "confluence_token_expired"
	ConfluenceErrorCodeRevoked       = "confluence_connection_revoked"
	ConfluenceErrorCodeUpstream      = "confluence_upstream_error"
)

type ConfluenceError struct {
	Category    string `json:"category"`
	Code        string `json:"code"`
	HTTPStatus  int    `json:"http_status,omitempty"`
	RetryAfter  string `json:"retry_after,omitempty"`
	Operation   string `json:"operation,omitempty"`
	UserMessage string `json:"message"`
	cause       error
}

func (err *ConfluenceError) Error() string {
	if err == nil {
		return ""
	}
	if strings.TrimSpace(err.UserMessage) != "" {
		return strings.TrimSpace(err.UserMessage)
	}
	if strings.TrimSpace(err.Code) != "" {
		return strings.TrimSpace(err.Code)
	}
	return "Confluence 요청을 완료하지 못했습니다."
}

func (err *ConfluenceError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.cause
}

func NewConfluenceValidationError(code string, message string) *ConfluenceError {
	return &ConfluenceError{
		Category:    ConfluenceErrorCategoryValidation,
		Code:        strings.TrimSpace(code),
		HTTPStatus:  400,
		UserMessage: strings.TrimSpace(message),
		cause:       ErrInvalidInput,
	}
}

func NewConfluenceConflictError(code string, message string) *ConfluenceError {
	return &ConfluenceError{
		Category:    ConfluenceErrorCategoryConflict,
		Code:        strings.TrimSpace(code),
		HTTPStatus:  409,
		UserMessage: strings.TrimSpace(message),
		cause:       ErrConflict,
	}
}

func NewConfluenceHTTPError(status int, retryAfter string, operation string) *ConfluenceError {
	category := ConfluenceErrorCategoryUpstream
	code := ConfluenceErrorCodeUpstream
	message := "Confluence 요청을 완료하지 못했습니다. 연결 상태와 권한을 확인하세요."
	switch status {
	case 401:
		category = ConfluenceErrorCategoryAuth
		code = ConfluenceErrorCodeUnauthorized
		message = "Confluence 인증이 만료되었거나 유효하지 않습니다. 연결을 다시 인증하세요."
	case 403:
		category = ConfluenceErrorCategoryPermission
		code = ConfluenceErrorCodeForbidden
		message = "Confluence 권한 또는 OAuth scope가 부족합니다. 연결 권한과 페이지 접근 권한을 확인하세요."
	case 404:
		category = ConfluenceErrorCategoryNotFound
		code = ConfluenceErrorCodeNotFound
		message = "Confluence 사이트 또는 페이지를 찾을 수 없습니다. cloud id와 page id를 확인하세요."
	case 429:
		category = ConfluenceErrorCategoryRateLimited
		code = ConfluenceErrorCodeRateLimited
		message = "Confluence 요청이 제한되었습니다. 잠시 후 다시 시도하세요."
	}
	if status >= 500 {
		message = "Confluence 서비스가 요청을 처리하지 못했습니다. 잠시 후 다시 시도하세요."
	}
	return &ConfluenceError{
		Category:    category,
		Code:        code,
		HTTPStatus:  status,
		RetryAfter:  strings.TrimSpace(retryAfter),
		Operation:   strings.TrimSpace(operation),
		UserMessage: message,
	}
}

func NewConfluenceTransportError(operation string, cause error) *ConfluenceError {
	return &ConfluenceError{
		Category:    ConfluenceErrorCategoryUpstream,
		Code:        ConfluenceErrorCodeUpstream,
		HTTPStatus:  502,
		Operation:   strings.TrimSpace(operation),
		UserMessage: "Confluence 요청을 보내지 못했습니다. 연결 상태와 권한을 확인하세요.",
		cause:       cause,
	}
}

func ConfluenceErrorDetails(err error) (*ConfluenceError, bool) {
	var confluenceErr *ConfluenceError
	if errors.As(err, &confluenceErr) && confluenceErr != nil {
		return confluenceErr, true
	}
	return nil, false
}

func ConfluenceErrorStatus(err error) int {
	if confluenceErr, ok := ConfluenceErrorDetails(err); ok && confluenceErr.HTTPStatus > 0 {
		return confluenceErr.HTTPStatus
	}
	if errors.Is(err, ErrInvalidInput) {
		return 400
	}
	if errors.Is(err, ErrConflict) {
		return 409
	}
	return 500
}

func ConfluenceSafeErrorMessage(err error) string {
	if confluenceErr, ok := ConfluenceErrorDetails(err); ok {
		return confluenceErr.Error()
	}
	return "Confluence 요청을 완료하지 못했습니다. 연결 상태와 권한을 확인하세요."
}

func ConfluenceHTTPErrorString(status int, operation string) string {
	operation = strings.TrimSpace(operation)
	if operation == "" {
		return fmt.Sprintf("confluence connector request returned %d", status)
	}
	return fmt.Sprintf("confluence connector %s returned %d", operation, status)
}
