package provider

import (
	"os"
	"path/filepath"
)

func init() {
	Register("claude", func(baseDir string) Provider {
		if baseDir == "" {
			home, _ := os.UserHomeDir()
			baseDir = filepath.Join(home, ".claude", "skills")
		}
		return &skillMDProvider{providerName: "claude", baseDir: baseDir}
	})
}
