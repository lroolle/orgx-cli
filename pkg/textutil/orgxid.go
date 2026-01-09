package textutil

import "regexp"

// IDPattern defines what constitutes a valid orgx ID.
// Used by both parser (extraction) and writer (existence check).
// Allows: UUIDs, org-roam timestamps, simple alphanumeric IDs.
const IDPattern = `[A-Za-z0-9_-]+`

// OrgxIDMarkerRe matches a complete orgx-id HTML comment marker.
// Anchored to match the entire line (with optional surrounding whitespace).
var OrgxIDMarkerRe = regexp.MustCompile(`^\s*<!--\s*orgx-id:\s*(` + IDPattern + `)\s*-->\s*$`)

// OrgxIDExtractRe extracts the ID from an orgx-id marker anywhere in text.
// Not anchored - used for finding markers in content.
var OrgxIDExtractRe = regexp.MustCompile(`<!--\s*orgx-id:\s*(` + IDPattern + `)\s*-->`)
