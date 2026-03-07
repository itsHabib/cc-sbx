// Package sync provides engines for syncing and diffing skills across providers.
package sync

import (
	"fmt"

	"github.com/user/skill-sync/internal/provider"
)

// SyncStatus represents the outcome of syncing a single skill to a target.
type SyncStatus string

const (
	// SyncSuccess indicates the skill was synced successfully.
	SyncSuccess SyncStatus = "success"
	// SyncError indicates the skill failed to sync.
	SyncError SyncStatus = "error"
	// SyncSkipped indicates the skill was skipped because it already exists in the target.
	SyncSkipped SyncStatus = "skipped"
)

// SyncDetail records the result of syncing one skill to one target.
type SyncDetail struct {
	SkillName string
	// Target is the provider name the skill was synced to.
	Target string
	Status SyncStatus
	// Error is non-nil only when Status is SyncError.
	Error error
}

// SyncResult aggregates the outcome of a sync operation.
type SyncResult struct {
	TotalSynced  int
	TotalSkipped int
	TotalErrored int
	// Details contains one entry per (skill, target) pair attempted.
	Details []SyncDetail
}

// SyncEngine orchestrates reading skills from a source provider and writing
// them to one or more target providers.
type SyncEngine struct {
	source  provider.Provider
	targets []provider.Provider
}

// NewSyncEngine creates a sync engine with a source and target providers.
func NewSyncEngine(source provider.Provider, targets []provider.Provider) *SyncEngine {
	return &SyncEngine{
		source:  source,
		targets: targets,
	}
}

// Sync reads skills from the source and writes them to all targets.
// If skillFilter is non-empty, only skills whose names appear in the filter are synced.
// If force is false, skills that already exist in a target are skipped.
// Returns a fatal error only if the source ListSkills call fails.
// Per-skill and per-target errors are captured in SyncResult.Details.
func (e *SyncEngine) Sync(skillFilter []string, force bool) (*SyncResult, error) {
	skills, err := e.source.ListSkills()
	if err != nil {
		return nil, fmt.Errorf("sync: list source skills: %w", err)
	}

	if len(skillFilter) > 0 {
		filterSet := make(map[string]bool, len(skillFilter))
		for _, name := range skillFilter {
			filterSet[name] = true
		}
		filtered := skills[:0:0]
		for _, s := range skills {
			if filterSet[s.Name] {
				filtered = append(filtered, s)
			}
		}
		skills = filtered
	}

	// Pre-load existing skill names per target for skip-existing behavior.
	targetExisting := make(map[string]map[string]bool, len(e.targets))
	if !force {
		for _, target := range e.targets {
			existing, listErr := target.ListSkills()
			if listErr != nil {
				// If we can't list, treat as empty (will attempt writes).
				targetExisting[target.Name()] = map[string]bool{}
				continue
			}
			nameSet := make(map[string]bool, len(existing))
			for _, s := range existing {
				nameSet[s.Name] = true
			}
			targetExisting[target.Name()] = nameSet
		}
	}

	result := &SyncResult{}

	for _, skill := range skills {
		full, err := e.source.ReadSkill(skill.Name)
		if err != nil {
			for _, target := range e.targets {
				detail := SyncDetail{
					SkillName: skill.Name,
					Target:    target.Name(),
					Status:    SyncError,
					Error:     fmt.Errorf("sync: read source skill %q: %w", skill.Name, err),
				}
				result.Details = append(result.Details, detail)
				result.TotalErrored++
			}
			continue
		}

		for _, target := range e.targets {
			// Skip if target already has this skill and force is off.
			if !force {
				if existing, ok := targetExisting[target.Name()]; ok && existing[skill.Name] {
					result.Details = append(result.Details, SyncDetail{
						SkillName: skill.Name,
						Target:    target.Name(),
						Status:    SyncSkipped,
					})
					result.TotalSkipped++
					continue
				}
			}

			if writeErr := target.WriteSkill(*full); writeErr != nil {
				result.Details = append(result.Details, SyncDetail{
					SkillName: skill.Name,
					Target:    target.Name(),
					Status:    SyncError,
					Error:     fmt.Errorf("sync: write skill %q to %q: %w", skill.Name, target.Name(), writeErr),
				})
				result.TotalErrored++
			} else {
				result.Details = append(result.Details, SyncDetail{
					SkillName: skill.Name,
					Target:    target.Name(),
					Status:    SyncSuccess,
				})
				result.TotalSynced++
			}
		}
	}

	return result, nil
}
