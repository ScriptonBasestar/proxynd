package ansible

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		wantMajor  int
		wantMinor  int
		wantPatch  int
		wantPrerel string
		wantBuild  string
		wantErr    bool
	}{
		{
			name:      "simple version",
			version:   "1.0.0",
			wantMajor: 1, wantMinor: 0, wantPatch: 0,
		},
		{
			name:      "version with prerelease",
			version:   "1.0.0-beta1",
			wantMajor: 1, wantMinor: 0, wantPatch: 0,
			wantPrerel: "beta1",
		},
		{
			name:      "version with build",
			version:   "1.0.0+20250124",
			wantMajor: 1, wantMinor: 0, wantPatch: 0,
			wantBuild: "20250124",
		},
		{
			name:      "version with prerelease and build",
			version:   "2.1.0-rc.1+build.123",
			wantMajor: 2, wantMinor: 1, wantPatch: 0,
			wantPrerel: "rc.1",
			wantBuild:  "build.123",
		},
		{
			name:    "invalid version",
			version: "invalid",
			wantErr: true,
		},
		{
			name:    "version with v prefix",
			version: "v1.0.0",
			wantErr: true,
		},
		{
			name:    "empty version",
			version: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ver, err := ParseVersion(tt.version)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantMajor, ver.Major)
			assert.Equal(t, tt.wantMinor, ver.Minor)
			assert.Equal(t, tt.wantPatch, ver.Patch)
			assert.Equal(t, tt.wantPrerel, ver.Prerelease)
			assert.Equal(t, tt.wantBuild, ver.Build)
			assert.Equal(t, tt.version, ver.String())
		})
	}
}

func TestVersion_Compare(t *testing.T) {
	tests := []struct {
		name    string
		v1      string
		v2      string
		wantCmp int
	}{
		{
			name: "equal versions",
			v1:   "1.0.0", v2: "1.0.0",
			wantCmp: 0,
		},
		{
			name: "v1 greater - major",
			v1:   "2.0.0", v2: "1.0.0",
			wantCmp: 1,
		},
		{
			name: "v1 less - major",
			v1:   "1.0.0", v2: "2.0.0",
			wantCmp: -1,
		},
		{
			name: "v1 greater - minor",
			v1:   "1.1.0", v2: "1.0.0",
			wantCmp: 1,
		},
		{
			name: "v1 less - minor",
			v1:   "1.0.0", v2: "1.1.0",
			wantCmp: -1,
		},
		{
			name: "v1 greater - patch",
			v1:   "1.0.1", v2: "1.0.0",
			wantCmp: 1,
		},
		{
			name: "v1 less - patch",
			v1:   "1.0.0", v2: "1.0.1",
			wantCmp: -1,
		},
		{
			name: "prerelease vs release",
			v1:   "1.0.0", v2: "1.0.0-beta1",
			wantCmp: 1, // Release > prerelease
		},
		{
			name: "prerelease comparison",
			v1:   "1.0.0-beta2", v2: "1.0.0-beta1",
			wantCmp: 1,
		},
		{
			name: "build metadata ignored",
			v1:   "1.0.0+build1", v2: "1.0.0+build2",
			wantCmp: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ver1, err := ParseVersion(tt.v1)
			require.NoError(t, err)

			ver2, err := ParseVersion(tt.v2)
			require.NoError(t, err)

			cmp := ver1.Compare(ver2)
			assert.Equal(t, tt.wantCmp, cmp)
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name    string
		v1      string
		v2      string
		wantCmp int
		wantErr bool
	}{
		{
			name: "valid comparison",
			v1:   "2.0.0", v2: "1.0.0",
			wantCmp: 1,
		},
		{
			name:    "invalid v1",
			v1:      "invalid", v2: "1.0.0",
			wantErr: true,
		},
		{
			name:    "invalid v2",
			v1:      "1.0.0", v2: "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmp, err := CompareVersions(tt.v1, tt.v2)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantCmp, cmp)
		})
	}
}

func TestMatchesVersionRange(t *testing.T) {
	tests := []struct {
		name         string
		version      string
		versionRange string
		wantMatch    bool
		wantErr      bool
	}{
		{
			name: "wildcard always matches",
			version: "1.2.3", versionRange: "*",
			wantMatch: true,
		},
		{
			name: "empty range always matches",
			version: "1.2.3", versionRange: "",
			wantMatch: true,
		},
		{
			name: "exact match ==",
			version: "1.0.0", versionRange: "==1.0.0",
			wantMatch: true,
		},
		{
			name: "exact match implicit ==",
			version: "1.0.0", versionRange: "1.0.0",
			wantMatch: true,
		},
		{
			name: "not equal !=",
			version: "1.0.1", versionRange: "!=1.0.0",
			wantMatch: true,
		},
		{
			name: "greater than >",
			version: "2.0.0", versionRange: ">1.0.0",
			wantMatch: true,
		},
		{
			name: "greater than or equal >=",
			version: "1.0.0", versionRange: ">=1.0.0",
			wantMatch: true,
		},
		{
			name: "less than <",
			version: "0.9.0", versionRange: "<1.0.0",
			wantMatch: true,
		},
		{
			name: "less than or equal <=",
			version: "1.0.0", versionRange: "<=1.0.0",
			wantMatch: true,
		},
		{
			name: "compatible release ~=",
			version: "1.2.5", versionRange: "~=1.2.0",
			wantMatch: true, // 1.2.5 >= 1.2.0 and < 1.3.0
		},
		{
			name: "compatible release mismatch minor",
			version: "1.3.0", versionRange: "~=1.2.0",
			wantMatch: false,
		},
		{
			name: "compatible release mismatch major",
			version: "2.0.0", versionRange: "~=1.2.0",
			wantMatch: false,
		},
		{
			name:    "invalid version",
			version: "invalid", versionRange: ">=1.0.0",
			wantErr: true,
		},
		{
			name:    "invalid range",
			version: "1.0.0", versionRange: ">>invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := MatchesVersionRange(tt.version, tt.versionRange)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantMatch, match)
		})
	}
}

func TestFindLatestVersion(t *testing.T) {
	tests := []struct {
		name        string
		versions    []string
		wantLatest  string
		wantErr     bool
	}{
		{
			name:       "single version",
			versions:   []string{"1.0.0"},
			wantLatest: "1.0.0",
		},
		{
			name:       "multiple versions",
			versions:   []string{"1.0.0", "2.0.0", "1.5.0"},
			wantLatest: "2.0.0",
		},
		{
			name:       "versions with prerelease",
			versions:   []string{"1.0.0-beta", "1.0.0", "0.9.0"},
			wantLatest: "1.0.0", // Release > prerelease
		},
		{
			name:       "unsorted versions",
			versions:   []string{"1.2.0", "1.10.0", "1.1.0"},
			wantLatest: "1.10.0",
		},
		{
			name:    "empty list",
			versions: []string{},
			wantErr: true,
		},
		{
			name:       "versions with invalid entries (skipped)",
			versions:   []string{"1.0.0", "invalid", "2.0.0"},
			wantLatest: "2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			latest, err := FindLatestVersion(tt.versions)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantLatest, latest)
		})
	}
}
