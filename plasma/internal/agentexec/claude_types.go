package agentexec

import (
	"time"
)

// ClaudeExecutor는 Claude CLI 프로세스를 실행하는 구체 어댑터 설정이다.
type ClaudeExecutor struct {
	Command      string
	WorkDir      string
	Model        string
	Timeout      time.Duration
	Env          []string
	Permission   string
	MaxBudgetUSD string
	MCPServer    ClaudeMCPServer
}

// ClaudeMCPServer는 Claude CLI에 넘길 MCP server 실행 명령과 인자다.
type ClaudeMCPServer struct {
	Name    string
	Command string
	Args    []string
}
