//go:build enterprise && !cloud

package app

// getEdition returns the current edition (enterprise)
func getEdition() string {
	return "enterprise"
}
