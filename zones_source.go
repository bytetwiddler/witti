package witti

import "fmt"

func zoneCandidates(zoneinfoRoot string) ([]string, string, error) {
	if zoneinfoRoot == "" {
		return defaultZoneNames, "built-in IANA zone list", nil
	}

	zones, err := collectZones(zoneinfoRoot)
	if err != nil {
		return nil, "", err
	}

	return zones, fmt.Sprintf("zoneinfo root %s", zoneinfoRoot), nil
}
