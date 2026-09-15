package articleexperiment

import (
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// VerifyIssueLockReceipt validates live-comment metadata and exact bundle
// digests supplied by a command adapter after it fetches Issue #461.
func VerifyIssueLockReceipt(loaded LoadedRealProtocol, receipt IssueLockReceipt, comment LiveIssueComment) error {
	if comment.Author != loaded.Protocol.IssueLock.Owner || receipt.Owner != comment.Author || receipt.CommentURL != comment.URL || receipt.CommentID != comment.ID || receipt.CommentCreatedAt != comment.CreatedAt {
		return fmt.Errorf("%w: Issue 461 live comment metadata differs", producterror.ErrConflict)
	}
	if !utf8.Valid(comment.Body) || bytesSHA256(comment.Body) != receipt.CommentBodySHA256 || !commentContainsBundleDigests(comment.Body, receipt) {
		return fmt.Errorf("%w: Issue 461 comment body does not bind the frozen bundle", producterror.ErrConflict)
	}
	if receipt.Repository != loaded.Protocol.IssueLock.Repository || receipt.Issue != loaded.Protocol.IssueLock.Issue || receipt.Owner != loaded.Protocol.IssueLock.Owner || strings.TrimSpace(receipt.CommentID) == "" || !validSHA256(receipt.CommentBodySHA256) || receipt.ProtocolSHA256 != loaded.ProtocolSHA256 || receipt.BlindSHA256 != loaded.BlindSHA256 || !reflect.DeepEqual(receipt.FixtureSHA256, loaded.fixtureSHA256) || !reflect.DeepEqual(receipt.ArmSHA256, loaded.armSHA256) {
		return fmt.Errorf("%w: Issue 461 lock receipt differs from frozen bundle", producterror.ErrConflict)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, receipt.CommentCreatedAt)
	if err != nil {
		return fmt.Errorf("%w: Issue 461 comment time is invalid", producterror.ErrInvalidInput)
	}
	protocolCreatedAt, _ := time.Parse(time.RFC3339Nano, loaded.Protocol.CreatedAt)
	if createdAt.Before(protocolCreatedAt) {
		return fmt.Errorf("%w: Issue 461 lock predates the protocol", producterror.ErrConflict)
	}
	commentURL, err := url.Parse(receipt.CommentURL)
	if err != nil || commentURL.Scheme != "https" || commentURL.Host != "github.com" || commentURL.Fragment != "issuecomment-"+receipt.CommentID {
		return fmt.Errorf("%w: Issue 461 comment URL is invalid", producterror.ErrInvalidInput)
	}
	wantPath := "/" + receipt.Repository + "/issues/" + strconv.Itoa(receipt.Issue)
	if commentURL.Path != wantPath {
		return fmt.Errorf("%w: Issue 461 comment URL does not identify the frozen issue", producterror.ErrConflict)
	}
	return nil
}

func commentContainsBundleDigests(commentBody []byte, receipt IssueLockReceipt) bool {
	body := string(commentBody)
	digests := []string{receipt.ProtocolSHA256, receipt.BlindSHA256}
	for _, values := range []map[string]string{receipt.FixtureSHA256, receipt.ArmSHA256} {
		keys := make([]string, 0, len(values))
		for key := range values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			digests = append(digests, values[key])
		}
	}
	for _, digest := range digests {
		if !validSHA256(digest) || strings.Count(body, digest) != 1 {
			return false
		}
	}
	return true
}
