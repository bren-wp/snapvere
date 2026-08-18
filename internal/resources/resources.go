package resources

import _ "embed"

// Icon is the single canonical Snapvera application icon used by the app,
// installer and uninstaller. Keeping one source asset prevents silent drift.
//
//go:embed snapvera.ico
var Icon []byte
