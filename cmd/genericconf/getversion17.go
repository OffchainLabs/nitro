//go:build !go1.18

package genericconf

func GetVersion() (string, string) {
	return "development", "development"
}
