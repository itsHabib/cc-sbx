package provider

import (
	"os"
	"path/filepath"
)

func init() {
	Register("copilot", func(baseDir string) Provider {
		if baseDir == "" {
			home, _ := os.UserHomeDir()
			baseDir = filepath.Join(home, ".copilot", "skills")
		}
		return &skillMDProvider{providerName: "copilot", baseDir: baseDir}
	})
}
