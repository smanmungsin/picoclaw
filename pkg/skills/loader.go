package skills

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"

	"github.com/sipeed/picoclaw/pkg/logger"
)

type SkillMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type SkillsLoader struct {
	workspace       string
	workspaceSkills string // workspace skills (项目级别)
	globalSkills    string // 全局 skills (~/.picoclaw/skills)
	builtinSkills   string // 内置 skills

	watcher         *fsnotify.Watcher
	watchMu         sync.Mutex
	hotReloadActive bool
}

type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Source      string `json:"source"`
}

func (info SkillInfo) validate() error {
	var errs error
	if info.Name == "" {
		errs = fmt.Errorf("name is required")
	} else {
		if len(info.Name) > MaxNameLength {
			errs = fmt.Errorf("name exceeds %d characters", MaxNameLength)
		}
		if !namePattern.MatchString(info.Name) {
			errs = fmt.Errorf("name must be alphanumeric with hyphens")
		}
	}
	if info.Description == "" {
		errs = fmt.Errorf("description is required")
	} else if len(info.Description) > MaxDescriptionLength {
		errs = fmt.Errorf("description exceeds %d character", MaxDescriptionLength)
	}
	return errs
}

var namePattern = regexp.MustCompile(`^[a-zA-Z0-9]+(-[a-zA-Z0-9]+)*$`)

const (
	MaxNameLength        = 64
	MaxDescriptionLength = 1024
)

func NewSkillsLoader(workspace string, globalSkills string, builtinSkills string) *SkillsLoader {
	sl := &SkillsLoader{
		workspace:       workspace,
		workspaceSkills: filepath.Join(workspace, "skills"),
		globalSkills:    globalSkills, // ~/.picoclaw/skills
		builtinSkills:   builtinSkills,
	}
	// Enable hot-reloading by default
	if err := sl.StartHotReloading(); err != nil {
		slog.Warn("Hot-reloading could not be started", "error", err)
	}
	return sl
}

// StartHotReloading enables hot-reloading for skills and modules
func (sl *SkillsLoader) StartHotReloading() error {
	sl.watchMu.Lock()
	defer sl.watchMu.Unlock()
	if sl.hotReloadActive {
		return nil // Already active
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	skillDirs := []string{}
	if sl.workspaceSkills != "" {
		skillDirs = append(skillDirs, sl.workspaceSkills)
	}
	if sl.globalSkills != "" {
		skillDirs = append(skillDirs, sl.globalSkills)
	}
	if sl.builtinSkills != "" {
		skillDirs = append(skillDirs, sl.builtinSkills)
	}
	for _, dir := range skillDirs {
		if err := w.Add(dir); err != nil {
			slog.Warn("Failed to watch skill dir", "dir", dir, "error", err)
		}
	}
	sl.watcher = w
	sl.hotReloadActive = true
	go sl.handleHotReloadEvents()
	return nil
}

// handleHotReloadEvents processes fsnotify events and reloads skills
func (sl *SkillsLoader) handleHotReloadEvents() {
	for {
		select {
		case event, ok := <-sl.watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) != 0 {
				slog.Info("Hot-reload: skill file changed", "file", event.Name, "op", event.Op.String())
				sl.ReloadSkills()
			}
		case err, ok := <-sl.watcher.Errors:
			if !ok {
				return
			}
			slog.Warn("Hot-reload watcher error", "error", err)
		}
	}
}

// ReloadSkills reloads all skills (can be extended to reload only changed ones)
func (sl *SkillsLoader) ReloadSkills() {
	slog.Info("Reloading skills due to change detected")
	// This can be extended to reload only changed skills
	// For now, just refresh the skill list
	_ = sl.ListSkills()
	sl.notifyAgent("Skills hot-reloaded")
}

// MergeSkillsFromPeer merges skills from a peer's skills summary (XML format expected)
func (sl *SkillsLoader) MergeSkillsFromPeer(peerSkills string) {
	// Parse peerSkills summary (expecting XML-like format)
	// This is a simple implementation: extract <name> and <description> and add if missing
	lines := strings.Split(peerSkills, "\n")
	var name, desc string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "<name>") && strings.HasSuffix(line, "</name>") {
			name = strings.TrimSuffix(strings.TrimPrefix(line, "<name>"), "</name>")
		}
		if strings.HasPrefix(line, "<description>") && strings.HasSuffix(line, "</description>") {
			desc = strings.TrimSuffix(strings.TrimPrefix(line, "<description>"), "</description>")
		}
		if name != "" && desc != "" {
			// Check if skill already exists
			exists := false
			for _, s := range sl.ListSkills() {
				if s.Name == name {
					exists = true
					break
				}
			}
			if !exists {
				// Add skill metadata to workspace (as a stub, just log)
				fmt.Printf("[SkillsLoader] Merged skill from peer: %s - %s\n", name, desc)
				// Real implementation: create SKILL.md or update metadata
			}
			name, desc = "", "" // Reset for next skill
		}
	}
}

// Health check: verify skills directory and auto-repair if missing
func (sl *SkillsLoader) HealthCheckAndRepair() {
	if sl.workspace == "" {
		logger.ErrorC("skills", "Workspace not set, attempting recovery")
		sl.workspace = os.TempDir()
	}
	skillsDir := filepath.Join(sl.workspace, "skills")
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		err := os.MkdirAll(skillsDir, 0o755)
		if err != nil {
			logger.ErrorCF("skills", "Failed to create skills directory", map[string]any{"error": err.Error()})
			sl.notifyAgent("CRITICAL: Skills directory could not be created")
		} else {
			sl.notifyAgent("Created default skills directory")
		}
	}
}

// notifyAgent sends a notification to the agent for critical recovery events
func (sl *SkillsLoader) notifyAgent(message string) {
	logger.WarnCF("skills", "Agent notification", map[string]any{"message": message})
	// Optionally send to agent bus if available (stub)
}

func (sl *SkillsLoader) ListSkills() []SkillInfo {
	skills := make([]SkillInfo, 0)

	if sl.workspaceSkills != "" {
		if dirs, err := os.ReadDir(sl.workspaceSkills); err == nil {
			for _, dir := range dirs {
				if dir.IsDir() {
					skillFile := filepath.Join(sl.workspaceSkills, dir.Name(), "SKILL.md")
					if _, err := os.Stat(skillFile); err == nil {
						info := SkillInfo{
							Name:   dir.Name(),
							Path:   skillFile,
							Source: "workspace",
						}
						skills = append(skills, info)
					}
				}
			}
		}
	}

	// 全局 skills (~/.picoclaw/skills) - 被 workspace skills 覆盖
	if sl.globalSkills != "" {
		if dirs, err := os.ReadDir(sl.globalSkills); err == nil {
			for _, dir := range dirs {
				if dir.IsDir() {
					skillFile := filepath.Join(sl.globalSkills, dir.Name(), "SKILL.md")
					if _, err := os.Stat(skillFile); err == nil {
						// 检查是否已被 workspace skills 覆盖
						exists := false
						for _, s := range skills {
							if s.Name == dir.Name() && s.Source == "workspace" {
								exists = true
								break
							}
						}
						if exists {
							continue
						}
						info := SkillInfo{
							Name:   dir.Name(),
							Path:   skillFile,
							Source: "global",
						}
						metadata := sl.getSkillMetadata(skillFile)
						if metadata != nil {
							info.Description = metadata.Description
							info.Name = metadata.Name
						}
						if err := info.validate(); err != nil {
							slog.Warn("invalid skill from global", "name", info.Name, "error", err)
							continue
						}
						skills = append(skills, info)
					}
				}
			}
		}
	}

	if sl.builtinSkills != "" {
		if dirs, err := os.ReadDir(sl.builtinSkills); err == nil {
			for _, dir := range dirs {
				if dir.IsDir() {
					skillFile := filepath.Join(sl.builtinSkills, dir.Name(), "SKILL.md")
					if _, err := os.Stat(skillFile); err == nil {
						// 检查是否已被 workspace 或 global skills 覆盖
						exists := false
						for _, s := range skills {
							if s.Name == dir.Name() && (s.Source == "workspace" || s.Source == "global") {
								exists = true
								break
							}
						}
						if exists {
							continue
						}
						info := SkillInfo{
							Name:   dir.Name(),
							Path:   skillFile,
							Source: "builtin",
						}
						metadata := sl.getSkillMetadata(skillFile)
						if metadata != nil {
							info.Description = metadata.Description
							info.Name = metadata.Name
						}
						if err := info.validate(); err != nil {
							slog.Warn("invalid skill from builtin", "name", info.Name, "error", err)
							continue
						}
						skills = append(skills, info)
					}
				}
			}
		}
	}

	return skills
}

func (sl *SkillsLoader) LoadSkill(name string) (string, bool) {
	// 1. 优先从 workspace skills 加载（项目级别）
	if sl.workspaceSkills != "" {
		skillFile := filepath.Join(sl.workspaceSkills, name, "SKILL.md")
		if content, err := os.ReadFile(skillFile); err == nil {
			return sl.stripFrontmatter(string(content)), true
		}
	}

	// 2. 其次从全局 skills 加载 (~/.picoclaw/skills)
	if sl.globalSkills != "" {
		skillFile := filepath.Join(sl.globalSkills, name, "SKILL.md")
		if content, err := os.ReadFile(skillFile); err == nil {
			return sl.stripFrontmatter(string(content)), true
		}
	}

	// 3. 最后从内置 skills 加载
	if sl.builtinSkills != "" {
		skillFile := filepath.Join(sl.builtinSkills, name, "SKILL.md")
		if content, err := os.ReadFile(skillFile); err == nil {
			return sl.stripFrontmatter(string(content)), true
		}
	}

	return "", false
}

func (sl *SkillsLoader) LoadSkillsForContext(skillNames []string) string {
	if len(skillNames) == 0 {
		return ""
	}

	var parts []string
	for _, name := range skillNames {
		content, ok := sl.LoadSkill(name)
		if ok {
			parts = append(parts, fmt.Sprintf("### Skill: %s\n\n%s", name, content))
		}
	}

	return strings.Join(parts, "\n\n---\n\n")
}

func (sl *SkillsLoader) BuildSkillsSummary() string {
	allSkills := sl.ListSkills()
	if len(allSkills) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "<skills>")
	for _, s := range allSkills {
		escapedName := escapeXML(s.Name)
		escapedDesc := escapeXML(s.Description)
		escapedPath := escapeXML(s.Path)

		lines = append(lines, fmt.Sprintf("    <name>%s</name>", escapedName))
		lines = append(lines, fmt.Sprintf("    <description>%s</description>", escapedDesc))
		lines = append(lines, fmt.Sprintf("    <location>%s</location>", escapedPath))
		lines = append(lines, fmt.Sprintf("    <source>%s</source>", s.Source))
		lines = append(lines, "  </skill>")
	}
	lines = append(lines, "</skills>")

	return strings.Join(lines, "\n")
}

func (sl *SkillsLoader) getSkillMetadata(skillPath string) *SkillMetadata {
	content, err := os.ReadFile(skillPath)
	if err != nil {
		return nil
	}

	frontmatter := sl.extractFrontmatter(string(content))
	if frontmatter == "" {
		return &SkillMetadata{
			Name: filepath.Base(filepath.Dir(skillPath)),
		}
	}

	// Try JSON first (for backward compatibility)
	var jsonMeta struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(frontmatter), &jsonMeta); err == nil {
		return &SkillMetadata{
			Name:        jsonMeta.Name,
			Description: jsonMeta.Description,
		}
	}

	// Fall back to simple YAML parsing
	yamlMeta := sl.parseSimpleYAML(frontmatter)
	return &SkillMetadata{
		Name:        yamlMeta["name"],
		Description: yamlMeta["description"],
	}
}

// parseSimpleYAML parses simple key: value YAML format
// Example: name: github\n description: "..."
func (sl *SkillsLoader) parseSimpleYAML(content string) map[string]string {
	result := make(map[string]string)

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Remove quotes if present
			value = strings.Trim(value, "\"'")
			result[key] = value
		}
	}

	return result
}

func (sl *SkillsLoader) extractFrontmatter(content string) string {
	// (?s) enables DOTALL mode so . matches newlines
	// Match first ---, capture everything until next --- on its own line
	re := regexp.MustCompile(`(?s)^---\n(.*)\n---`)
	match := re.FindStringSubmatch(content)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func (sl *SkillsLoader) stripFrontmatter(content string) string {
	re := regexp.MustCompile(`^---\n.*?\n---\n`)
	return re.ReplaceAllString(content, "")
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
