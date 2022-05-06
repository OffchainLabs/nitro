//go:build !go1.18

package conf

func GetVersion() (string, string) {
	return "development", "development"
}
