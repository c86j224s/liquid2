package agentexec

import (
	"strconv"
	"strings"
)

// codexCommandArgs는 요청 설정을 실제로 해석하는 CLI command에 결합한다.
// resume은 자체 option parser를 가지므로 model과 effort override는 부모 exec가
// 아니라 resume subcommand 뒤에 위치해야 한다.
func codexCommandArgs(server CodexMCPServer, req AgentRequest, workDir, lastPath string) []string {
	args := []string{"exec"}
	resumed := strings.TrimSpace(req.PreviousSessionID) != ""
	if resumed {
		args = append(args, "resume")
	}
	if req.EphemeralSession {
		args = append(args, "--ephemeral")
	}
	if req.IgnoreUserConfig {
		args = append(args, "--ignore-user-config")
	}
	if model := strings.TrimSpace(req.Model); model != "" {
		args = append(args, "--model", model)
	}
	if effort := strings.TrimSpace(req.ReasoningEffort); effort != "" {
		args = append(args, "-c", "model_reasoning_effort="+strconv.Quote(effort))
	}
	args = append(args, "--json")
	args = append(args, codexMCPConfigArgs(server, req)...)

	if resumed {
		return append(args,
			"-c", `sandbox_mode="read-only"`,
			"--skip-git-repo-check",
			"--ignore-rules",
			"--output-last-message", lastPath,
			strings.TrimSpace(req.PreviousSessionID),
			"-",
		)
	}
	return append(args,
		"--sandbox", "read-only",
		"--skip-git-repo-check",
		"--ignore-rules",
		"-C", workDir,
		"--output-last-message", lastPath,
		"-",
	)
}
