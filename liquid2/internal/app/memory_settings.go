package app

func (reader memoryReader) Settings() AppSettings {
	return cloneAppSettings(reader.state.settings)
}

func (tx memoryTx) PutSettings(settings AppSettings) {
	tx.state.settings = cloneAppSettings(settings)
}
