package mcpdocs

import (
	"embed"
	"strings"
)

const (
	// MIMETypeMarkdown is the wire MIME type used for every static MCP
	// documentation resource.
	MIMETypeMarkdown = "text/markdown"

	URITools     = "plasma://docs/mcp/tools"
	URIErrors    = "plasma://docs/mcp/errors"
	URIReporting = "plasma://docs/mcp/reporting"
	URISources   = "plasma://docs/mcp/sources"
	URIMermaid   = "plasma://docs/mcp/mermaid"
)

// Resource is the public metadata advertised by resources/list.
//
// URI is the stable identity of the document. The package owns only static
// documentation resources, so MIMEType is currently always text/markdown.
type Resource struct {
	URI         string
	Name        string
	Description string
	MIMEType    string
}

// Document is a static Markdown resource returned by Read.
type Document struct {
	Resource
	Text string
}

type catalogEntry struct {
	Resource
	filename string
}

//go:embed tools.md errors.md reporting.md sources.md mermaid.md
var resourceFS embed.FS

var catalog = []catalogEntry{
	{
		Resource: Resource{
			URI:         URITools,
			Name:        "Plasma MCP Tools",
			Description: "Static guide to Plasma MCP tool groups and calling boundaries.",
			MIMEType:    MIMETypeMarkdown,
		},
		filename: "tools.md",
	},
	{
		Resource: Resource{
			URI:         URIErrors,
			Name:        "Plasma MCP Errors",
			Description: "Static guide to protocol errors, tool errors, retry signals, and safe reporting.",
			MIMEType:    MIMETypeMarkdown,
		},
		filename: "errors.md",
	},
	{
		Resource: Resource{
			URI:         URIReporting,
			Name:        "Plasma MCP Reporting",
			Description: "Static guide to report planning, patching, assembly, editing, and finalization boundaries.",
			MIMEType:    MIMETypeMarkdown,
		},
		filename: "reporting.md",
	},
	{
		Resource: Resource{
			URI:         URISources,
			Name:        "Plasma MCP Sources",
			Description: "Static guide to source snapshots, source candidates, connector search, and live local path observation.",
			MIMEType:    MIMETypeMarkdown,
		},
		filename: "sources.md",
	},
	{
		Resource: Resource{
			URI:         URIMermaid,
			Name:        "Plasma MCP Mermaid",
			Description: "Static guide to Mermaid diagram preflight and reporting use.",
			MIMEType:    MIMETypeMarkdown,
		},
		filename: "mermaid.md",
	},
}

// List returns the deterministic static resource catalog.
func List() []Resource {
	resources := make([]Resource, 0, len(catalog))
	for _, entry := range catalog {
		resources = append(resources, entry.Resource)
	}
	return resources
}

// Read returns one static Markdown resource by stable URI.
func Read(uri string) (Document, bool) {
	trimmed := strings.TrimSpace(uri)
	for _, entry := range catalog {
		if entry.URI != trimmed {
			continue
		}
		content, err := resourceFS.ReadFile(entry.filename)
		if err != nil {
			return Document{}, false
		}
		return Document{Resource: entry.Resource, Text: string(content)}, true
	}
	return Document{}, false
}
