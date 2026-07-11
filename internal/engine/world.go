package engine

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/crimsab/oneday/internal/storage"
)

// KnownLocation represents a discovered location stored in world_state.known_locations_json.
// The JSON may be a plain string array OR an array of objects depending on how it was written.
type KnownLocation struct {
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	DiscoveredTurn int    `json:"discovered_turn,omitempty"`
	Region         string `json:"region,omitempty"`
}

// parseKnownLocations parses KnownLocationsJSON which may be:
//   - a JSON array of strings: ["Loc A", "Loc B"]
//   - a JSON array of KnownLocation objects
//   - empty / null / "[]"
func parseKnownLocations(raw string) []KnownLocation {
	if raw == "" || raw == "null" || raw == "[]" {
		return nil
	}
	// Try array of objects first.
	var objs []KnownLocation
	if err := json.Unmarshal([]byte(raw), &objs); err == nil && len(objs) > 0 && objs[0].Name != "" {
		return objs
	}
	// Fall back to array of strings.
	var strs []string
	if err := json.Unmarshal([]byte(raw), &strs); err == nil {
		result := make([]KnownLocation, 0, len(strs))
		for _, s := range strs {
			if s != "" {
				result = append(result, KnownLocation{Name: s})
			}
		}
		return result
	}
	return nil
}

// FormatMapView builds a text representation of discovered locations.
func FormatMapView(world *storage.WorldState) string {
	if world == nil {
		return "No world state available."
	}

	locs := parseKnownLocations(world.KnownLocationsJSON)

	var sb strings.Builder
	sb.WriteString("=== World Map ===\n")

	if world.CurrentLocation != "" {
		sb.WriteString(fmt.Sprintf("\nCurrent Location: >> %s <<\n", world.CurrentLocation))
	}

	if len(locs) == 0 {
		sb.WriteString("\nNo locations discovered yet. Explore the world!\n")
		return sb.String()
	}

	sb.WriteString("\nDiscovered Locations:\n")

	// Group by region if regions are present.
	byRegion := map[string][]KnownLocation{}
	var regionOrder []string
	for _, loc := range locs {
		region := loc.Region
		if region == "" {
			region = "Unknown Region"
		}
		if _, exists := byRegion[region]; !exists {
			regionOrder = append(regionOrder, region)
		}
		byRegion[region] = append(byRegion[region], loc)
	}

	hasRegions := len(regionOrder) > 1 || (len(regionOrder) == 1 && regionOrder[0] != "Unknown Region")

	if hasRegions {
		for _, region := range regionOrder {
			sb.WriteString(fmt.Sprintf("\n  [%s]\n", region))
			for _, loc := range byRegion[region] {
				isCurrent := strings.EqualFold(loc.Name, world.CurrentLocation)
				marker := "  * "
				if isCurrent {
					marker = "  > "
				}
				line := marker + loc.Name
				if isCurrent {
					line += " [current]"
				}
				if loc.Description != "" {
					line += fmt.Sprintf("\n      %s", loc.Description)
				}
				sb.WriteString(line + "\n")
			}
		}
	} else {
		for _, loc := range locs {
			isCurrent := strings.EqualFold(loc.Name, world.CurrentLocation)
			marker := "  * "
			if isCurrent {
				marker = "  > "
			}
			line := marker + loc.Name
			if isCurrent {
				line += " [current]"
			}
			if loc.Description != "" {
				line += fmt.Sprintf("\n      %s", loc.Description)
			}
			sb.WriteString(line + "\n")
		}
	}

	return sb.String()
}

// AddLocationToWorldState adds a location to world.KnownLocationsJSON if not already present.
// Handles both string-array and object-array formats transparently.
func AddLocationToWorldState(world *storage.WorldState, locationName string, currentTurn int) bool {
	if locationName == "" {
		return false
	}
	locs := parseKnownLocations(world.KnownLocationsJSON)

	// Check for duplicate (case-insensitive).
	for _, l := range locs {
		if strings.EqualFold(l.Name, locationName) {
			return false
		}
	}

	// Determine storage format: if existing entries have descriptions/regions → use object format.
	useObjects := false
	for _, l := range locs {
		if l.Description != "" || l.Region != "" || l.DiscoveredTurn != 0 {
			useObjects = true
			break
		}
	}

	if useObjects {
		locs = append(locs, KnownLocation{
			Name:           locationName,
			DiscoveredTurn: currentTurn,
		})
		if b, err := json.Marshal(locs); err == nil {
			world.KnownLocationsJSON = string(b)
		}
	} else {
		// Simple string array.
		var strs []string
		for _, l := range locs {
			strs = append(strs, l.Name)
		}
		strs = append(strs, locationName)
		if b, err := json.Marshal(strs); err == nil {
			world.KnownLocationsJSON = string(b)
		}
	}
	return true
}
