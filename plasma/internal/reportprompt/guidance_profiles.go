package reportprompt

const (
	// 과거 report event, artifact, API replay, 실험 재현성을 위해 legacy 실험 profile은
	// 여기서 계속 받아들인다. 사용자에게 어떤 active profile을 보여 줄지는 Web UI가 소유한다.
	reportGenerationGuidanceProfileG2                            = "g2"
	reportGenerationGuidanceProfileNone                          = "none"
	reportGenerationGuidanceProfileSectionContract               = "section-contract"
	reportGenerationGuidanceProfileSectionContractCoverage       = "section-contract-coverage"
	reportGenerationGuidanceProfileSectionIntent                 = "section-intent"
	reportGenerationGuidanceProfileSourceClusterFirst            = "source-cluster-first"
	reportGenerationGuidanceProfileSectionBrief                  = "section-brief"
	reportGenerationGuidanceProfileSectionBriefCluster           = "section-brief-cluster-memory"
	reportGenerationGuidanceProfileSectionBriefVisualPlan        = "section-brief-visual-plan"
	reportGenerationGuidanceProfileSectionBriefClusterVisualPlan = "section-brief-cluster-memory-visual-plan"
	reportGenerationGuidanceProfilePlanReview                    = "plan-review"
	reportGenerationGuidanceProfilePartAssemblyEditTools         = "part-assembly-edit-tools"
	reportGenerationGuidanceProfileVisualSupplement              = "visual-supplement"
	reportGenerationGuidanceProfileVisualPlan                    = "visual-plan"
	reportGenerationGuidanceProfileDefault                       = reportGenerationGuidanceProfileNarrativeContract
	reportGenerationGuidanceProfileLongFormDefault               = reportGenerationGuidanceProfileSectionBriefClusterNarrativeContract
)
