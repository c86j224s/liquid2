package app

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
)

func renderMarkdownDocument(tx RepositoryReader, record documentRecord) []byte {
	var body bytes.Buffer
	fmt.Fprintf(&body, "# %s\n\n", markdownHeading(record.meta.Title))
	writeMetadata(&body, record, documentTags(tx, record))
	for _, content := range record.contents {
		writeContent(&body, content)
	}
	if len(record.blobs) > 0 {
		body.WriteString("## Attachments\n\n")
		for _, blob := range record.blobs {
			fmt.Fprintf(&body, "- `%s` (%s, %d bytes)\n", blob.Filename, blob.MimeType, blob.Size)
		}
		body.WriteByte('\n')
	}
	return body.Bytes()
}

func writeMetadata(body *bytes.Buffer, record documentRecord, tags []Tag) {
	meta := record.meta
	fmt.Fprintf(body, "- ID: `%s`\n", meta.ID)
	fmt.Fprintf(body, "- Kind: `%s`\n", meta.Kind)
	if meta.SourceURL != nil {
		fmt.Fprintf(body, "- Source URL: <%s>\n", *meta.SourceURL)
	}
	if meta.CanonicalURL != nil {
		fmt.Fprintf(body, "- Canonical URL: <%s>\n", *meta.CanonicalURL)
	}
	if meta.Language != nil {
		fmt.Fprintf(body, "- Language: `%s`\n", *meta.Language)
	}
	if len(tags) > 0 {
		names := make([]string, 0, len(tags))
		for _, tag := range tags {
			names = append(names, tag.Name)
		}
		fmt.Fprintf(body, "- Tags: %s\n", strings.Join(names, ", "))
	}
	body.WriteByte('\n')
}

func writeContent(body *bytes.Buffer, content DocumentContent) {
	heading := content.Role
	if content.Language != nil {
		heading += " " + *content.Language
	}
	fmt.Fprintf(body, "## %s\n\n", markdownHeading(heading))
	switch content.Format {
	case ContentFormatMarkdown:
		body.WriteString(strings.TrimRight(content.Content, "\n"))
		body.WriteString("\n\n")
	case ContentFormatHTML:
		body.WriteString("```html\n")
		body.WriteString(fenceSafe(content.Content))
		body.WriteString("\n```\n\n")
	default:
		body.WriteString("```\n")
		body.WriteString(fenceSafe(content.Content))
		body.WriteString("\n```\n\n")
	}
}

func markdownHeading(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.TrimSpace(value)
	if value == "" {
		return "Untitled"
	}
	return value
}

func fenceSafe(value string) string {
	return strings.ReplaceAll(strings.TrimRight(value, "\n"), "```", "` ` `")
}

func sanitizeExportName(value string) string {
	value = strings.TrimSpace(value)
	var builder strings.Builder
	for _, item := range value {
		switch {
		case item >= 'a' && item <= 'z':
			builder.WriteRune(item)
		case item >= 'A' && item <= 'Z':
			builder.WriteRune(item)
		case item >= '0' && item <= '9':
			builder.WriteRune(item)
		case item == '-' || item == '_' || item == '.':
			builder.WriteRune(item)
		case unicode.IsSpace(item):
			builder.WriteByte('_')
		default:
			builder.WriteByte('_')
		}
	}
	name := strings.Trim(builder.String(), "._")
	if name == "" {
		return "item"
	}
	return name
}

func exportBlobName(blob BlobMetadata) string {
	name := sanitizeExportName(blob.Filename)
	if name == "item" {
		return sanitizeExportName(blob.ID)
	}
	return sanitizeExportName(blob.ID) + "-" + name
}
