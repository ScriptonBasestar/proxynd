//go:build enterprise && !cloud

package routers

// getEdition returns the current edition (enterprise)
func getEdition() string {
	return "enterprise"
}
