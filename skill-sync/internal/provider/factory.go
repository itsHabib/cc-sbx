package provider

import (
	"os"
	"path/filepath"
)

func init() {
	Register("factory", func(baseDir string) Provider {
		if baseDir == "" {
			home, _ := os.UserHomeDir()
			baseDir = filepath.Join(home, ".factory", "skills")
		}
		return &skillMDProvider{providerName: "factory", baseDir: baseDir}
	})
}
