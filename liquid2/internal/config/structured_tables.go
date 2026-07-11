package config

func (c Config) applyAPIGroupTables(values map[string]any) {
	c.applyAPIServerTable(namedTable(values, "server"))
	c.applyAPIPathsTable(namedTable(values, "paths"))
	c.applyAPILoggingTable(namedTable(values, "logging"))
	c.applyAPIRuntimeTable(namedTable(values, "runtime"))
	c.applyAPITranslationTable(namedTable(values, "translation"))
}

func (c Config) applyAPIServerTable(values map[string]any) {
	if addr, ok := combinedAddrValue(values); ok {
		c.values[KeyAddr] = addr
	}
	c.setValue(KeyCORSOrigins, values["cors_origins"])
}

func (c Config) applyAPIPathsTable(values map[string]any) {
	c.setValue(KeyDBPath, values["db_path"])
	c.setValue(KeyExportDir, values["export_dir"])
	c.setValue(KeyBackupDir, values["backup_dir"])
}

func (c Config) applyAPILoggingTable(values map[string]any) {
	c.setValue(KeyLogLevel, values["level"])
	c.setValue(KeyLogFormat, values["format"])
	c.setValue(KeyLogSource, values["source"])
}

func (c Config) applyAPIRuntimeTable(values map[string]any) {
	c.setValue(KeyJobsEnabled, values["jobs_enabled"])
	c.setValue(KeySeedDemo, values["seed_demo"])
}

func (c Config) applyAPITranslationTable(values map[string]any) {
	c.setValue(KeyTranslationProvider, values["provider"])
	c.setValue(KeyCodexCommand, values["codex_command"])
	c.setValue(KeyCodexModel, values["codex_model"])
	c.setValue(KeyCodexTimeoutSeconds, values["codex_timeout_seconds"])

	codex := namedTable(values, "codex")
	c.setValue(KeyCodexCommand, codex["command"])
	c.setValue(KeyCodexModel, codex["model"])
	c.setValue(KeyCodexTimeoutSeconds, codex["timeout_seconds"])
}

func (c Config) applyWebTable(values map[string]any) {
	c.setValue(KeyWebAddr, values["addr"])
	c.setValue(KeyWebAddr, values["web_addr"])
	c.setValue(KeyWebPort, values["port"])
	c.setValue(KeyWebPort, values["web_port"])
	c.setValue(KeyEnvironmentLabel, values["environment_label"])
}

func (c Config) applyWebGroupTables(values map[string]any) {
	c.applyWebServerTable(namedTable(values, "server"))
	c.applyWebRuntimeTable(namedTable(values, "runtime"))
}

func (c Config) applyWebServerTable(values map[string]any) {
	c.setValue(KeyWebAddr, values["addr"])
	c.setValue(KeyWebPort, values["port"])
	c.setValue(KeyEnvironmentLabel, values["environment_label"])
}

func (c Config) applyWebRuntimeTable(values map[string]any) {
	c.setValue(KeyEnvironmentLabel, values["environment_label"])
}

func (c Config) setValue(key string, raw any) {
	if value, ok := configValue(raw); ok {
		c.values[key] = value
	}
}
