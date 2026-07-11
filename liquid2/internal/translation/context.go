package translation

import "github.com/c86j224s/liquid2/internal/app"

type sourceContext struct {
	Payload TranslateDocumentPayload
	Source  app.DocumentContent
}

type translatedContext struct {
	Source sourceContext
	Result Result
}
