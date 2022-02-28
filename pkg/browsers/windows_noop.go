//go:build !windows

package browsers

func HandleWindowsBrowserSearch() (string, error) {
	return "", nil
}
