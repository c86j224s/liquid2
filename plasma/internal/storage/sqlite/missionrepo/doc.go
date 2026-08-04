// Package missionrepo persists missions, ledger events, and mission projections.
//
// It owns SQL for the mission capability only. Cross-capability orchestration and
// transaction ownership remain in the parent sqlite package, which calls the narrow
// exported transaction helpers here when a root-level workflow spans repositories.
package missionrepo
