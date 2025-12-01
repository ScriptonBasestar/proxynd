package ansible

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Semantic version pattern: MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]
var semverRegex = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

// Version represents a semantic version
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Build      string
	Original   string
}

// ParseVersion parses a semantic version string
func ParseVersion(version string) (*Version, error) {
	if version == "" {
		return nil, ErrInvalidVersion
	}

	matches := semverRegex.FindStringSubmatch(version)
	if matches == nil {
		return nil, ErrInvalidVersion
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	return &Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: matches[4],
		Build:      matches[5],
		Original:   version,
	}, nil
}

// String returns the string representation of the version
func (v *Version) String() string {
	return v.Original
}

// Compare compares two versions
// Returns: -1 if v < other, 0 if v == other, 1 if v > other
func (v *Version) Compare(other *Version) int {
	// Compare major
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}

	// Compare minor
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}

	// Compare patch
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}

	// Compare prerelease
	// No prerelease > with prerelease
	if v.Prerelease == "" && other.Prerelease != "" {
		return 1
	}
	if v.Prerelease != "" && other.Prerelease == "" {
		return -1
	}

	if v.Prerelease != other.Prerelease {
		return comparePrerelease(v.Prerelease, other.Prerelease)
	}

	// Versions are equal (build metadata is ignored in comparison)
	return 0
}

// comparePrerelease compares prerelease versions
func comparePrerelease(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	minLen := len(aParts)
	if len(bParts) < minLen {
		minLen = len(bParts)
	}

	for i := 0; i < minLen; i++ {
		aNum, aErr := strconv.Atoi(aParts[i])
		bNum, bErr := strconv.Atoi(bParts[i])

		// Both are numbers
		if aErr == nil && bErr == nil {
			if aNum != bNum {
				if aNum < bNum {
					return -1
				}
				return 1
			}
			continue
		}

		// Numeric < alphanumeric
		if aErr == nil && bErr != nil {
			return -1
		}
		if aErr != nil && bErr == nil {
			return 1
		}

		// Both are alphanumeric
		if aParts[i] != bParts[i] {
			if aParts[i] < bParts[i] {
				return -1
			}
			return 1
		}
	}

	// Shorter prerelease < longer prerelease
	if len(aParts) != len(bParts) {
		if len(aParts) < len(bParts) {
			return -1
		}
		return 1
	}

	return 0
}

// ValidateVersion validates a semantic version string
func ValidateVersion(version string) error {
	_, err := ParseVersion(version)
	return err
}

// CompareVersions compares two version strings
func CompareVersions(v1, v2 string) (int, error) {
	ver1, err := ParseVersion(v1)
	if err != nil {
		return 0, err
	}

	ver2, err := ParseVersion(v2)
	if err != nil {
		return 0, err
	}

	return ver1.Compare(ver2), nil
}

// MatchesVersionRange checks if a version matches a version range specification
// Supports: >=1.0.0, >1.0.0, <=1.0.0, <1.0.0, ==1.0.0, !=1.0.0, ~=1.0.0
func MatchesVersionRange(version, versionRange string) (bool, error) {
	ver, err := ParseVersion(version)
	if err != nil {
		return false, err
	}

	// Parse version range
	versionRange = strings.TrimSpace(versionRange)

	// Handle special cases
	if versionRange == "*" || versionRange == "" {
		return true, nil
	}

	// Extract operator and target version
	var operator string
	var targetVersionStr string

	if strings.HasPrefix(versionRange, ">=") {
		operator = ">="
		targetVersionStr = strings.TrimSpace(versionRange[2:])
	} else if strings.HasPrefix(versionRange, "<=") {
		operator = "<="
		targetVersionStr = strings.TrimSpace(versionRange[2:])
	} else if strings.HasPrefix(versionRange, "~=") {
		operator = "~="
		targetVersionStr = strings.TrimSpace(versionRange[2:])
	} else if strings.HasPrefix(versionRange, "!=") {
		operator = "!="
		targetVersionStr = strings.TrimSpace(versionRange[2:])
	} else if strings.HasPrefix(versionRange, "==") {
		operator = "=="
		targetVersionStr = strings.TrimSpace(versionRange[2:])
	} else if strings.HasPrefix(versionRange, ">") {
		operator = ">"
		targetVersionStr = strings.TrimSpace(versionRange[1:])
	} else if strings.HasPrefix(versionRange, "<") {
		operator = "<"
		targetVersionStr = strings.TrimSpace(versionRange[1:])
	} else {
		// No operator, assume ==
		operator = "=="
		targetVersionStr = versionRange
	}

	targetVer, err := ParseVersion(targetVersionStr)
	if err != nil {
		return false, fmt.Errorf("%w: %s", ErrInvalidVersionRange, versionRange)
	}

	cmp := ver.Compare(targetVer)

	switch operator {
	case "==":
		return cmp == 0, nil
	case "!=":
		return cmp != 0, nil
	case ">":
		return cmp > 0, nil
	case ">=":
		return cmp >= 0, nil
	case "<":
		return cmp < 0, nil
	case "<=":
		return cmp <= 0, nil
	case "~=":
		// Compatible release: ~=1.2.3 means >=1.2.3, <1.3.0
		if ver.Major != targetVer.Major || ver.Minor != targetVer.Minor {
			return false, nil
		}
		return cmp >= 0, nil
	default:
		return false, fmt.Errorf("%w: unknown operator %s", ErrInvalidVersionRange, operator)
	}
}

// FindLatestVersion finds the latest version from a list of version strings
func FindLatestVersion(versions []string) (string, error) {
	if len(versions) == 0 {
		return "", ErrVersionNotFound
	}

	latest := versions[0]
	latestVer, err := ParseVersion(latest)
	if err != nil {
		return "", err
	}

	for i := 1; i < len(versions); i++ {
		ver, err := ParseVersion(versions[i])
		if err != nil {
			continue // Skip invalid versions
		}

		if ver.Compare(latestVer) > 0 {
			latest = versions[i]
			latestVer = ver
		}
	}

	return latest, nil
}
