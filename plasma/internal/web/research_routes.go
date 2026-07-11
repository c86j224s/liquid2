package web

import (
	"net/http"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/researchproposal"
	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidates"
)

func (server *Server) handleMissionRecords(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	records, err := server.collectRecords(r.Context(), missionID, nil)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, records)
}

func (server *Server) handleMissionClaims(w http.ResponseWriter, r *http.Request, missionID string, rest []string) {
	if len(rest) != 2 || rest[1] != "confidence" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req claimConfidenceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	event, err := server.service.UpdateClaimConfidence(r.Context(), app.UpdateClaimConfidenceRequest{
		EventID:   newID("evt"),
		MissionID: missionID,
		ClaimID:   rest[0],
		Confidence: app.Confidence{
			Level:             req.Level,
			Rationale:         req.Rationale,
			OpenRisks:         req.OpenRisks,
			NeedsVerification: req.NeedsVerification,
		},
		BasisEvidenceIDs: req.BasisEvidenceIDs,
		Origin:           "user",
		Producer:         app.Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	detail, err := server.missionDetail(r.Context(), missionID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"event": event, "detail": detail})
}

func (server *Server) handleMissionCandidates(w http.ResponseWriter, r *http.Request, missionID string, rest []string) {
	if len(rest) == 2 && rest[0] == "sources" && rest[1] == "reject" {
		server.handleRejectSourceCandidate(w, r, missionID)
		return
	}
	if len(rest) == 2 && rest[0] == "sources" && rest[1] == "restore" {
		server.handleRestoreSourceCandidate(w, r, missionID)
		return
	}
	if len(rest) != 1 || rest[0] != "evidence" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req evidenceCandidateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	summary := strings.TrimSpace(req.Summary)
	if summary == "" {
		writeError(w, http.StatusBadRequest, "candidate summary is required")
		return
	}
	snapshotID := strings.TrimSpace(req.SnapshotID)
	artifactID := strings.TrimSpace(req.ArtifactID)
	if snapshotID == "" || artifactID == "" {
		writeError(w, http.StatusBadRequest, "candidate source snapshot and artifact are required")
		return
	}
	evidenceType := strings.TrimSpace(req.EvidenceType)
	if evidenceType == "" {
		evidenceType = "observation"
	}

	evidenceID := newID("evd")
	proposalID := newID("prp")
	evidenceEventID := newID("evt")
	proposalEventID := newID("evt")
	result, err := server.service.CreateEvidenceProposal(r.Context(), researchproposal.BuildManualEvidenceCandidateProposalRequest(researchproposal.ManualEvidenceCandidateProposalRequest{
		MissionID:       missionID,
		EvidenceID:      evidenceID,
		ProposalID:      proposalID,
		EvidenceEventID: evidenceEventID,
		ProposalEventID: proposalEventID,
		Summary:         summary,
		EvidenceType:    evidenceType,
		SnapshotID:      snapshotID,
		ArtifactID:      artifactID,
		Producer:        app.Producer{Type: "user", ID: "plasma-ui"},
	}))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (server *Server) handleRejectSourceCandidate(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req rejectSourceCandidateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	event, err := sourcecandidates.Reject(r.Context(), server.service, sourcecandidates.SourceCandidateDecisionRequest{
		EventID:   newID("evt"),
		MissionID: missionID,
		URL:       req.URL,
		Reason:    req.Reason,
		Producer:  app.Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"event": event})
}

func (server *Server) handleRestoreSourceCandidate(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req restoreSourceCandidateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	event, err := sourcecandidates.Restore(r.Context(), server.service, sourcecandidates.SourceCandidateDecisionRequest{
		EventID:   newID("evt"),
		MissionID: missionID,
		URL:       req.URL,
		Reason:    req.Reason,
		Producer:  app.Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"event": event})
}

func (server *Server) handleMissionProposals(w http.ResponseWriter, r *http.Request, missionID string, rest []string) {
	if len(rest) == 0 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		proposals, err := server.service.ListProposalBundles(r.Context(), missionID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"proposals": proposals})
		return
	}
	if len(rest) != 2 || (rest[1] != "approve" && rest[1] != "reject") {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	server.decideProposal(w, r, missionID, rest[0], rest[1])
}

func (server *Server) decideProposal(w http.ResponseWriter, r *http.Request, missionID string, proposalID string, action string) {
	proposal, err := server.service.GetProposalBundle(r.Context(), proposalID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if proposal.MissionID != missionID {
		writeError(w, http.StatusBadRequest, "proposal belongs to another mission")
		return
	}
	eventReq, nextState := researchproposal.BuildProposalDecisionAppendRequest(researchproposal.ProposalDecisionAppendRequest{
		EventID:  newID("evt"),
		Proposal: proposal,
		Action:   action,
		Producer: app.Producer{Type: "user", ID: "plasma-ui"},
	})
	event, err := server.service.AppendEvent(r.Context(), eventReq)
	if err != nil {
		writeAppError(w, err)
		return
	}
	updated, err := server.service.UpdateProposalBundleState(r.Context(), app.UpdateProposalBundleStateRequest{
		ProposalID:      proposal.ProposalID,
		State:           nextState,
		DecisionEventID: event.EventID,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"proposal": updated, "event": event})
}
