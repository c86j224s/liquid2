// Package sqlitevalue owns SQLite scalar encoding shared by focused repository packages.
//
// The functions in this package preserve the legacy root sqlite.Store time, JSON, and
// boolean conversion semantics so structural repository splits do not alter persisted
// values or error text.
package sqlitevalue
