# Third-Party Notices

This repository includes generated code and vendored browser libraries.

## Vendored Browser Libraries

- `plasma/internal/web/static/vendor/markdown-it.min.js`
  - Project: markdown-it
  - Version: 14.2.0
  - License: MIT
  - Source: https://github.com/markdown-it/markdown-it
- `plasma/internal/web/static/vendor/purify.min.js`
  - Project: DOMPurify
  - Version: 3.4.11
  - License: Apache License 2.0 or Mozilla Public License 2.0
  - Source: https://github.com/cure53/DOMPurify

## Generated Code

- `liquid2/client/api/`
  - Generated from the Liquid2 OpenAPI contract with OpenAPI Generator.
- `liquid2/internal/storage/sqlite/sqlc/`
  - Generated from SQL queries with sqlc.

Generated files remain part of this source tree for reproducible local builds.
Regenerate them from the product-owned source contracts rather than editing
generated output by hand.
