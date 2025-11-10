//go:build !enterprise && !cloud

package routers

// getEdition returns the current edition (core)
func getEdition() string {
	return "core"
}
