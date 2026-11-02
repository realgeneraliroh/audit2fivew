// Copyright (c) 2026 J
// All rights reserved.
// Released under the GNU Affero General Public License v3.0.

// Package avc provides parsing and representation of SELinux AVC denials.
package avc

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Denial represents a single parsed SELinux AVC denial event.
type Denial struct {
	ID          int64     // Unique primary key in the SQLite database.
	Timestamp   time.Time // Logged timestamp of the denial event.
	Serial      string    // Audit event serial number (e.g. "456").
	RawLog      string    // Original raw audit log line.
	Permissions []string  // Denied permissions (e.g. ["read", "write"]).
	TClass      string    // Target security class (e.g. "file", "dir", "process").
	SContext    string    // Full source SELinux security context.
	TContext    string    // Full target SELinux security context.
	PID         int       // Process ID that triggered the denial.
	Comm        string    // Executable/command name (e.g. "httpd").
	Name        string    // Target object basename (if reported by audit).
	Dev         string    // Storage device identifier.
	Ino         string    // Inode number of the target object.
	Path        string    // Full filesystem path of the target object (if reported).
	Permissive  bool      // True if the denial occurred in permissive mode (audit only).
	ContentHash string    // Stable SHA-256 deduplication hash based on semantic identity.
	FirstSeen   time.Time // First time this specific denial was recorded.
	LastSeen    time.Time // Most recent time this denial was recorded.
	Count       int       // Total number of occurrences observed.
}

// Pre-compiled patterns for parsing AVC denial lines.
var (
	// avcRe captures the audit timestamp, serial, and denied permissions.
	// Example: type=AVC msg=audit(1609459200.123:100): avc:  denied  { read write }
	avcRe = regexp.MustCompile(
		`type=AVC\s+msg=audit\((\d+)\.(\d+):(\d+)\):\s*avc:\s*denied\s*\{\s*([^}]+?)\s*\}`)

	// fieldRe captures key=value or key="quoted value" pairs.
	fieldRe = regexp.MustCompile(`\b(\w+)=("(?:[^"\\]|\\.)*"|\S+)`)
)

// Parse extracts an AVC denial from a single audit log line.
// Returns an error if the line is not a valid AVC denial.
func Parse(line string) (*Denial, error) {
	if !strings.Contains(line, "type=AVC") || !strings.Contains(line, "denied") {
		return nil, fmt.Errorf("not an AVC denial line")
	}

	m := avcRe.FindStringSubmatch(line)
	if m == nil {
		return nil, fmt.Errorf("failed to parse AVC header")
	}

	epochSec, _ := strconv.ParseInt(m[1], 10, 64)
	epochMs, _ := strconv.ParseInt(m[2], 10, 64)
	serial := m[3]
	perms := strings.Fields(strings.TrimSpace(m[4]))

	ts := time.Unix(epochSec, epochMs*int64(time.Millisecond))

	d := &Denial{
		Timestamp:   ts,
		Serial:      serial,
		RawLog:      line,
		Permissions: perms,
		FirstSeen:   ts,
		LastSeen:    ts,
		Count:       1,
	}

	// Extract all key=value fields from the line.
	for _, kv := range fieldRe.FindAllStringSubmatch(line, -1) {
		key := kv[1]
		val := strings.Trim(kv[2], `"`)
		switch key {
		case "pid":
			d.PID, _ = strconv.Atoi(val)
		case "comm":
			d.Comm = val
		case "name":
			d.Name = val
		case "dev":
			d.Dev = val
		case "ino":
			d.Ino = val
		case "path":
			d.Path = val
		case "scontext":
			d.SContext = val
		case "tcontext":
			d.TContext = val
		case "tclass":
			d.TClass = val
		case "permissive":
			d.Permissive = val == "1"
		}
	}

	d.ContentHash = d.computeHash()
	return d, nil
}

// computeHash creates a stable deduplication hash from the denial's
// semantic identity (contexts, class, permissions, command).
func (d *Denial) computeHash() string {
	data := fmt.Sprintf("%s|%s|%s|%s|%s",
		d.SContext, d.TContext, d.TClass,
		strings.Join(d.Permissions, ","), d.Comm)
	h := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", h[:16])
}

// PermissionsStr returns permissions as a space-separated string.
func (d *Denial) PermissionsStr() string {
	return strings.Join(d.Permissions, " ")
}

// SType extracts the SELinux type from the source context.
// For "system_u:system_r:httpd_t:s0" it returns "httpd_t".
func (d *Denial) SType() string {
	return extractType(d.SContext)
}

// TType extracts the SELinux type from the target context.
func (d *Denial) TType() string {
	return extractType(d.TContext)
}

// extractType pulls the type field (third colon-separated component)
// from an SELinux security context string.
func extractType(ctx string) string {
	parts := strings.Split(ctx, ":")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ctx
}

