//go:build !enterprise && !cloud

package app

// getEdition returns the current edition (core)
func getEdition() string {
	return "core"
}
