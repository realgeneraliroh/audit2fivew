// Copyright (c) 2026 J
// All rights reserved.
// Released under the GNU Affero General Public License v3.0.

// Package reader ingests AVC denial events from the audit subsystem,
// either by reading the raw log file directly or by shelling out to
// ausearch(8).
package reader

import (
	"bufio"
	"os"
	"os/exec"
	"strings"

	"audit2fivew/internal/avc"
)

// Read parses every AVC denial from the audit log at the given path.
// Lines that are not valid AVC denials are silently skipped.
func Read(path string) ([]*avc.Denial, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var denials []*avc.Denial
	scanner := bufio.NewScanner(f)
	// Audit lines can be long; raise the buffer limit.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		d, err := avc.Parse(scanner.Text())
		if err != nil {
			continue // not an AVC denial
		}
		denials = append(denials, d)
	}
	if err := scanner.Err(); err != nil {
		return denials, err // return what we have so far
	}
	return denials, nil
}

// ReadAusearch runs `ausearch -m avc --raw` and parses the output.
// It returns an error if ausearch is not installed or fails to execute.
func ReadAusearch() ([]*avc.Denial, error) {
	cmd := exec.Command("ausearch", "-m", "avc", "--raw")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var denials []*avc.Denial
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "----") {
			continue
		}
		d, err := avc.Parse(line)
		if err != nil {
			continue
		}
		denials = append(denials, d)
	}
	return denials, nil
}

