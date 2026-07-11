package config

func (c *Config) applyServerTable(values map[string]any) {
	c.applyTable(values)
	if addr, ok := combinedAddrValue(values); ok {
		c.Addr = addr
	}
}

func (c *Config) applyPathsTable(values map[string]any) {
	c.setString("db_path", stringValue(values["db_path"]))
	c.setString("static_dir", stringValue(values["static_dir"]))
}

func (c *Config) applyAgentTables(values map[string]any) {
	if value, ok := listValue(values["enabled"]); ok {
		c.setString("agent", value)
	}
	c.setString("agent_workdir", stringValue(values["workdir"]))
	c.setString("agent_timeout", stringValue(values["timeout"]))

	codex := namedTable(values, "codex")
	c.setString("codex_command", stringValue(codex["command"]))
	c.setString("workflow_goal_model", stringValue(codex["workflow_goal_model"]))
	c.setString("workflow_goal_reasoning_effort", stringValue(codex["workflow_goal_reasoning_effort"]))

	claude := namedTable(values, "claude")
	c.setString("claude_command", stringValue(claude["command"]))
	c.setString("claude_model", stringValue(claude["model"]))
	c.setString("claude_max_budget_usd", stringValue(claude["max_budget_usd"]))
}

func (c *Config) applyLocalSourcesTable(values map[string]any) {
	if roots, ok := values["roots"]; ok {
		c.setList(configList(roots))
	}
}

func (c *Config) applyConfluenceOAuthTable(values map[string]any) {
	c.setString("confluence_oauth_client_id", stringValue(values["client_id"]))
	c.setString("confluence_oauth_client_secret", stringValue(values["client_secret"]))
	c.setString("confluence_oauth_redirect_uri", stringValue(values["redirect_uri"]))
	c.setConfluenceOAuthScopes(configList(values["scopes"]))
	c.setString("confluence_oauth_authorize_url", stringValue(values["authorize_url"]))
	c.setString("confluence_oauth_token_url", stringValue(values["token_url"]))
	c.setString("confluence_oauth_discovery_url", stringValue(values["discovery_url"]))
}
