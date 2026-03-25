package witti

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// collectZones walks the zoneinfo root and returns relative zone names like "Europe/Paris".
func collectZones(root string) ([]string, error) {
	var zones []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				// Root itself is unreadable or missing; surface the error.
				return err
			}
			// Skip other unreadable entries silently.
			return nil
		}
		if d.IsDir() {
			return nil
		}
		// Skip obvious non-zone files
		name := d.Name()
		if strings.HasSuffix(name, ".tab") || strings.HasSuffix(name, ".zi") || strings.HasSuffix(name, ".txt") {
			return nil
		}
		// Resolve relative zone name (e.g., "Europe/Paris")
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		// Convert Windows separators to '/' for Go's LoadLocation.
		zoneName := filepath.ToSlash(rel)
		zones = append(zones, zoneName)
		return nil
	})
	return zones, err
}
