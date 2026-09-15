// Package researchinspection owns bounded research-object reads.
//
// It owns the read request/result contract, validation, byte-range chunking, and
// callback sequencing. The app package remains responsible for materializing
// payloads, including PDF/local-path observation, security and visibility policy,
// and report materialization.
package researchinspection
