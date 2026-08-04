package agentexec

import (
	"strconv"
	"strings"
)

func codexMCPConfigArgs(server CodexMCPServer, req AgentRequest) []string {
	if req.DisableTools {
		return nil
	}
	command := strings.TrimSpace(server.Command)
	if command == "" {
		return nil
	}
	server.Args = codexMCPArgsForRequest(server.Args, req)
	name := sanitizeMCPServerName(server.Name)
	args := []string{
		"-c", "mcp_servers." + name + ".command=" + tomlString(command),
		"-c", "mcp_servers." + name + ".args=" + tomlStringArray(server.Args),
		"-c", "mcp_servers." + name + ".enabled=true",
		"-c", "mcp_servers." + name + ".default_tools_approval_mode=" + tomlString("approve"),
	}
	if server.Required {
		args = append(args, "-c", "mcp_servers."+name+".required=true")
	}
	if server.StartupTimeoutSec > 0 {
		args = append(args, "-c", "mcp_servers."+name+".startup_timeout_sec="+strconv.Itoa(server.StartupTimeoutSec))
	}
	if server.ToolTimeoutSec > 0 {
		args = append(args, "-c", "mcp_servers."+name+".tool_timeout_sec="+strconv.Itoa(server.ToolTimeoutSec))
	}
	if enabledTools := effectiveMCPEnabledTools(server.EnabledTools, req.ExtraMCPTools, req.ReplaceMCPTools); len(enabledTools) > 0 {
		args = append(args, "-c", "mcp_servers."+name+".enabled_tools="+tomlStringArray(enabledTools))
	}
	return args
}

// CodexMCPConfigArgs returns Codex CLI MCP server configuration argv.
func CodexMCPConfigArgs(server CodexMCPServer, req AgentRequest) []string {
	return codexMCPConfigArgs(server, req)
}
