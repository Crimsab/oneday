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

func FormatCanonicalMapView(world *storage.PlayerWorldProjection) string {
	if world == nil {
		return "No world state available."
	}
	var sb strings.Builder
	sb.WriteString("=== World Map ===\n\n")
	sb.WriteString(fmt.Sprintf("Current Location: >> %s <<\n", world.CurrentLocation))
	sb.WriteString(fmt.Sprintf("World Time: %s\n", world.Clock.DisplayText))
	sb.WriteString(fmt.Sprintf("Weather: %s\n", world.Weather.Label))
	sb.WriteString("\nDiscovered Locations:\n")
	for _, loc := range world.Locations {
		marker := "  * "
		if loc.ID == world.CurrentLocationID {
			marker = "  > "
		}
		sb.WriteString(marker + loc.Name + "\n")
	}
	if len(world.Edges) > 0 {
		sb.WriteString("\nKnown Routes:\n")
		names := map[string]string{}
		for _, loc := range world.Locations {
			names[loc.ID] = loc.Name
		}
		for _, edge := range world.Edges {
			sb.WriteString(fmt.Sprintf("  %s -> %s (%d min", names[edge.FromLocationID], names[edge.ToLocationID], edge.TravelMinutes))
			if edge.Direction != "" {
				sb.WriteString(", " + edge.Direction)
			}
			sb.WriteString(")\n")
		}
	}
	return sb.String()
}

const (
	spatialContextRegionLimit   = 40
	spatialContextLocationLimit = 48
	spatialContextRouteLimit    = 64
)

// TravelPolicyFromSettingJSON keeps imported/story-pack travel rules useful
// without making older stories depend on a new setting shape.
func TravelPolicyFromSettingJSON(raw string) string {
	var setting struct {
		WorldRules struct {
			TravelMode string `json:"travel_mode"`
		} `json:"world_rules"`
	}
	if json.Unmarshal([]byte(raw), &setting) == nil && setting.WorldRules.TravelMode == "graph" {
		return "graph"
	}
	return "narrative"
}

func FormatSpatialNarratorContext(world *storage.PlayerWorldProjection, travelPolicy string) string {
	if world == nil {
		return ""
	}
	regionNames := make(map[string]string, len(world.Regions))
	for _, region := range world.Regions {
		regionNames[region.ID] = region.Name
	}
	locationNames := make(map[string]string, len(world.Locations))
	for _, location := range world.Locations {
		locationNames[location.ID] = location.Name
	}
	var lines []string
	lines = append(lines, "## Canonical spatial model")
	lines = append(lines, fmt.Sprintf("Current location: %s (id=%s)", world.CurrentLocation, world.CurrentLocationID))
	if travelPolicy == "graph" {
		lines = append(lines, "Travel policy: graph. Movement must use a known direct route, or location_transition must explicitly discover a justified new route.")
	} else {
		lines = append(lines, "Travel policy: narrative. New justified routes may be declared in location_transition when movement reveals them.")
	}
	if len(world.Regions) > 0 {
		lines = append(lines, "Known regions:")
		for _, region := range world.Regions[:min(len(world.Regions), spatialContextRegionLimit)] {
			parent := regionNames[region.ParentRegionID]
			if parent == "" {
				parent = "world"
			}
			lines = append(lines, fmt.Sprintf("- %s [%s], parent: %s", region.Name, region.Kind, parent))
		}
		if omitted := len(world.Regions) - spatialContextRegionLimit; omitted > 0 {
			lines = append(lines, fmt.Sprintf("- ... %d additional regions omitted", omitted))
		}
	}
	if len(world.Locations) > 0 {
		lines = append(lines, "Known locations:")
		for _, location := range world.Locations[:min(len(world.Locations), spatialContextLocationLimit)] {
			region := regionNames[location.RegionID]
			if region == "" {
				region = "world"
			}
			parent := locationNames[location.ParentLocationID]
			if parent == "" {
				parent = "none"
			}
			lines = append(lines, fmt.Sprintf("- %s [%s], region: %s, parent location: %s", location.Name, location.Kind, region, parent))
		}
		if omitted := len(world.Locations) - spatialContextLocationLimit; omitted > 0 {
			lines = append(lines, fmt.Sprintf("- ... %d additional locations omitted", omitted))
		}
	}
	if len(world.Edges) > 0 {
		lines = append(lines, "Known routes:")
		for _, edge := range world.Edges[:min(len(world.Edges), spatialContextRouteLimit)] {
			lines = append(lines, fmt.Sprintf("- %s -> %s, direction: %s, mode: %s, minutes: %d", locationNames[edge.FromLocationID], locationNames[edge.ToLocationID], edge.Direction, edge.TravelMode, edge.TravelMinutes))
		}
		if omitted := len(world.Edges) - spatialContextRouteLimit; omitted > 0 {
			lines = append(lines, fmt.Sprintf("- ... %d additional routes omitted", omitted))
		}
	}
	lines = append(lines, "Preserve these canonical names and relationships. Use location_transition for any spatial change or discovery.")
	return strings.Join(lines, "\n")
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
