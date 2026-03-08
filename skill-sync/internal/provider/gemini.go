package provider

import (
	"os"
	"path/filepath"
)

func init() {
	Register("gemini", func(baseDir string) Provider {
		if baseDir == "" {
			home, _ := os.UserHomeDir()
			baseDir = filepath.Join(home, ".gemini", "skills")
		}
		return &skillMDProvider{providerName: "gemini", baseDir: baseDir}
	})
}
