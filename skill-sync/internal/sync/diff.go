package sync

import (
	"fmt"
	"strings"

	"github.com/user/skill-sync/internal/provider"
)

// SkillDrift records the drift status of a single skill in a target provider.
type SkillDrift struct {
	SkillName string
	Status    provider.SkillStatus
	// UnifiedDiff contains a unified diff string. Only populated when Status is Modified;
	// empty for InSync, MissingInTarget, and ExtraInTarget.
	UnifiedDiff string
}

// DriftReport contains per-target drift results for all skills.
type DriftReport struct {
	// Results maps target provider name to the drift status of each skill in that target.
	Results map[string][]SkillDrift
}

// DetailedDiff contains unified diffs for modified skills in a specific target.
type DetailedDiff struct {
	TargetName string
	// Diffs contains only skills with Status Modified, each with UnifiedDiff populated.
	Diffs []SkillDrift
}

// DiffEngine compares skills between a source and one or more target providers.
type DiffEngine struct {
	source  provider.Provider
	targets []provider.Provider
}

// NewDiffEngine creates a diff engine with a source and target providers.
func NewDiffEngine(source provider.Provider, targets []provider.Provider) *DiffEngine {
	return &DiffEngine{
		source:  source,
		targets: targets,
	}
}

// normalizeContent trims trailing whitespace and newlines for comparison.
func normalizeContent(s string) string {
	return strings.TrimRight(s, " \t\r\n")
}

// Status compares source skills against all targets and returns a DriftReport
// with per-skill status for each target.
func (e *DiffEngine) Status() (*DriftReport, error) {
	sourceSkills, err := e.source.ListSkills()
	if err != nil {
		return nil, fmt.Errorf("diff: list source skills: %w", err)
	}

	report := &DriftReport{
		Results: make(map[string][]SkillDrift, len(e.targets)),
	}

	for _, target := range e.targets {
		drifts, err := e.compareTarget(sourceSkills, target)
		if err != nil {
			return nil, fmt.Errorf("diff: compare target %q: %w", target.Name(), err)
		}
		report.Results[target.Name()] = drifts
	}

	return report, nil
}

// compareTarget compares source skills against a single target provider.
func (e *DiffEngine) compareTarget(sourceSkills []provider.Skill, target provider.Provider) ([]SkillDrift, error) {
	targetSkills, err := target.ListSkills()
	if err != nil {
		return nil, fmt.Errorf("list target skills: %w", err)
	}

	targetMap := make(map[string]provider.Skill, len(targetSkills))
	for _, ts := range targetSkills {
		targetMap[ts.Name] = ts
	}

	sourceNames := make(map[string]bool, len(sourceSkills))
	var drifts []SkillDrift

	for _, ss := range sourceSkills {
		sourceNames[ss.Name] = true

		ts, exists := targetMap[ss.Name]
		if !exists {
			drifts = append(drifts, SkillDrift{
				SkillName: ss.Name,
				Status:    provider.MissingInTarget,
			})
			continue
		}

		srcFull, err := e.source.ReadSkill(ss.Name)
		if err != nil {
			return nil, fmt.Errorf("read source skill %q: %w", ss.Name, err)
		}
		tgtFull, err := target.ReadSkill(ts.Name)
		if err != nil {
			return nil, fmt.Errorf("read target skill %q: %w", ts.Name, err)
		}

		if normalizeContent(srcFull.Content) == normalizeContent(tgtFull.Content) {
			drifts = append(drifts, SkillDrift{
				SkillName: ss.Name,
				Status:    provider.InSync,
			})
		} else {
			diff := unifiedDiff(ss.Name, srcFull.Content, tgtFull.Content)
			drifts = append(drifts, SkillDrift{
				SkillName:   ss.Name,
				Status:      provider.Modified,
				UnifiedDiff: diff,
			})
		}
	}

	for _, ts := range targetSkills {
		if !sourceNames[ts.Name] {
			drifts = append(drifts, SkillDrift{
				SkillName: ts.Name,
				Status:    provider.ExtraInTarget,
			})
		}
	}

	return drifts, nil
}

// Diff returns detailed unified diffs for a specific target.
// Returns an error if targetName is not in the engine's target list.
func (e *DiffEngine) Diff(targetName string) (*DetailedDiff, error) {
	var target provider.Provider
	for _, t := range e.targets {
		if t.Name() == targetName {
			target = t
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("diff: unknown target %q", targetName)
	}

	sourceSkills, err := e.source.ListSkills()
	if err != nil {
		return nil, fmt.Errorf("diff: list source skills: %w", err)
	}

	drifts, err := e.compareTarget(sourceSkills, target)
	if err != nil {
		return nil, fmt.Errorf("diff: %w", err)
	}

	result := &DetailedDiff{
		TargetName: targetName,
	}
	for _, d := range drifts {
		if d.Status == provider.Modified {
			result.Diffs = append(result.Diffs, d)
		}
	}

	return result, nil
}

// unifiedDiff produces a unified diff string between two texts.
// Uses a simple line-by-line comparison with context lines.
func unifiedDiff(name, a, b string) string {
	aLines := splitLines(a)
	bLines := splitLines(b)

	// Compute LCS table
	lcs := computeLCS(aLines, bLines)

	// Build edit script from LCS
	type edit struct {
		op   byte // '=' keep, '-' delete, '+' insert
		line string
	}

	var edits []edit
	i, j := len(aLines), len(bLines)
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && aLines[i-1] == bLines[j-1] {
			edits = append(edits, edit{'=', aLines[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]) {
			edits = append(edits, edit{'+', bLines[j-1]})
			j--
		} else {
			edits = append(edits, edit{'-', aLines[i-1]})
			i--
		}
	}

	// Reverse edits (we built them backwards)
	for left, right := 0, len(edits)-1; left < right; left, right = left+1, right-1 {
		edits[left], edits[right] = edits[right], edits[left]
	}

	// Generate unified diff with 3 lines of context
	const contextLines = 3
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("--- a/%s\n", name))
	sb.WriteString(fmt.Sprintf("+++ b/%s\n", name))

	// Find hunks: groups of changes with context
	type hunk struct {
		startA, countA int
		startB, countB int
		lines          []edit
	}

	var hunks []hunk
	n := len(edits)

	inChange := func(idx int) bool {
		return idx >= 0 && idx < n && edits[idx].op != '='
	}

	visited := make([]bool, n)
	for idx := 0; idx < n; idx++ {
		if visited[idx] || !inChange(idx) {
			continue
		}

		// Find range of this hunk with context
		start := idx - contextLines
		if start < 0 {
			start = 0
		}

		// Find end of changes in this hunk, merging nearby changes
		end := idx
		for end < n {
			if inChange(end) {
				end++
				continue
			}
			// Check if another change is within context range
			nextChange := -1
			for k := end; k < n && k <= end+2*contextLines; k++ {
				if inChange(k) {
					nextChange = k
					break
				}
			}
			if nextChange >= 0 {
				end = nextChange + 1
			} else {
				break
			}
		}

		// Add trailing context
		hunkEnd := end + contextLines
		if hunkEnd > n {
			hunkEnd = n
		}

		// Count lines in A and B for this hunk
		aStart, bStart := 0, 0
		aPos, bPos := 0, 0
		for k := 0; k < start; k++ {
			switch edits[k].op {
			case '=':
				aPos++
				bPos++
			case '-':
				aPos++
			case '+':
				bPos++
			}
		}
		aStart = aPos + 1
		bStart = bPos + 1

		var hunkEdits []edit
		aCount, bCount := 0, 0
		for k := start; k < hunkEnd; k++ {
			visited[k] = true
			hunkEdits = append(hunkEdits, edits[k])
			switch edits[k].op {
			case '=':
				aCount++
				bCount++
			case '-':
				aCount++
			case '+':
				bCount++
			}
		}

		hunks = append(hunks, hunk{
			startA: aStart, countA: aCount,
			startB: bStart, countB: bCount,
			lines: hunkEdits,
		})
	}

	for _, h := range hunks {
		sb.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", h.startA, h.countA, h.startB, h.countB))
		for _, e := range h.lines {
			switch e.op {
			case '=':
				sb.WriteString(" " + e.line + "\n")
			case '-':
				sb.WriteString("-" + e.line + "\n")
			case '+':
				sb.WriteString("+" + e.line + "\n")
			}
		}
	}

	return sb.String()
}

// splitLines splits text into lines, handling different line endings.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	// Remove trailing empty string from split if original ended with newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// computeLCS builds the LCS length table for two string slices.
func computeLCS(a, b []string) [][]int {
	m, n := len(a), len(b)
	table := make([][]int, m+1)
	for i := range table {
		table[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				table[i][j] = table[i-1][j-1] + 1
			} else if table[i-1][j] >= table[i][j-1] {
				table[i][j] = table[i-1][j]
			} else {
				table[i][j] = table[i][j-1]
			}
		}
	}
	return table
}
