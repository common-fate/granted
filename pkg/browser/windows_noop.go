//go:build !windows

package browser

func HandleWindowsBrowserSearch() (string, error) {
	return "", nil
}
