// Package artifactrepo persists raw artifacts and source snapshots.
//
// Raw artifacts and source snapshots remain together because snapshot rows own ordered
// links to artifacts. Root sqlite.Store keeps cross-capability transaction ownership
// and calls only the narrow exported transaction helpers in this package.
package artifactrepo
