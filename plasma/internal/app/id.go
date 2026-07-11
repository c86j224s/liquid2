package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

func newAppID(prefix string) string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s_%s_%s", strings.TrimSuffix(prefix, "_"), time.Now().UTC().Format("20060102150405"), hex.EncodeToString(b[:]))
}
