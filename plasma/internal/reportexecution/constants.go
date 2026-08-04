package reportexecution

const (
	DefaultMode  = ModePlanned
	ModeOneTake  = "one_take"
	ModePlanned  = "planned"
	ModeLongForm = "long_form"

	DefaultSessionPolicy      = SessionPolicySameSession
	SessionPolicySameSession  = "same_session"
	SessionPolicyIsolatedFork = "isolated_fork"

	SessionPolicySelectionAutoIsolatedFork          = "auto_isolated_fork"
	SessionPolicySelectionAutoSameSessionNoSession  = "auto_same_session_no_pre_report_session"
	SessionPolicySelectionAutoSameSessionNoForker   = "auto_same_session_no_forkable_executor"
	SessionPolicySelectionAutoSameSessionForkFailed = "auto_same_session_fork_unavailable"
	SessionPolicySelectionAutoSameSessionOneTake    = "auto_same_session_one_take"
	SessionPolicySelectionExplicitIsolatedFork      = "explicit_isolated_fork"
	SessionPolicySelectionExplicitSameSession       = "explicit_same_session"

	labelOneTake  = "원테이크 보고서"
	labelPlanned  = "보고서"
	labelLongForm = "장문 보고서"

	DesignTargetDesigned = "designed_html"

	ExportKindSelfContainedHTML   = "self_contained_html_report_artifact"
	ExportKindDesignedHTML        = "designed_html_report_artifact"
	ExportKindHumanizedMarkdown   = "humanized_markdown_report_artifact"
	ExportTargetSelfContainedHTML = "self_contained_html"
	ExportTargetDesignedHTML      = "designed_html"
	ExportTargetHumanizedMarkdown = "humanized_markdown"
	DesignedContentModelContract  = "dh26_inline_images"
	HumanizeProfileH5             = "h5-full-report-tone-pass"
	HumanizeTransportPatch        = "mcp_patch"
)
