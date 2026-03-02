package brain

import (
	"sync"
	"time"
)

// MemoryModule is a generic interface for a memory module (short-term, long-term, etc.)
type MemoryModule interface {
	Remember(key string, value any) error
	Recall(key string) (any, error)
	Search(query string) ([]any, error)
	Forget(key string) error
	ListKeys() ([]string, error)
}

// Event represents a single event in the brain's timeline
type Event struct {
	Timestamp time.Time
	Type      string
	Data      any
}

// Brain is the main struct combining all memory modules and event sourcing
type Brain struct {
	ShortTerm     MemoryModule
	LongTerm      MemoryModule
	Timeline      []Event
	mu            sync.RWMutex
	SelfReflect   *SelfReflectionModule
	reflectEvery  int
	eventCount    int
	reflectOn     map[string]bool // event types that trigger reflection
	ReportFunc    func(summary string)
	modules       map[string]any // extensible modules (conversation, todo, etc.)
	Security      *SecurityModule
}

func NewBrain(shortTerm, longTerm MemoryModule) *Brain {
   // Survival: check memory modules and auto-repair if missing
   if shortTerm == nil {
	   // Attempt to recover short-term memory
	   shortTerm = NewInMemoryMemory()
   }
   if longTerm == nil {
	   // Attempt to recover long-term memory
	   longTerm = NewInMemoryMemory()
   }
   // Survival: check for required tools and auto-install if missing
	requiredTools := []string{"tmux", "summarize", "gh", "devmem", "curl", "wget", "git"}
   missingTools := []string{}
   for _, tool := range requiredTools {
	   if !isToolAvailable(tool) {
		   missingTools = append(missingTools, tool)
		   autoInstallTool(tool)
	   }
   }
   if len(missingTools) > 0 {
	   fmt.Printf("[Survival] Missing critical tools: %v. Attempted auto-install.\n", missingTools)
   }
// isToolAvailable checks if a binary is available in PATH
func isToolAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// autoInstallTool attempts to install a missing tool (stub: extend for real install)
func autoInstallTool(name string) {
	   fmt.Printf("[Survival] Attempting to auto-install tool: %s\n", name)
	   // Platform-specific install logic
	   var installCmd string
	   switch name {
	   case "tmux", "curl", "wget", "git", "gh", "devmem", "summarize":
		   if isWindows() {
			   // Use winget or choco for Windows
			   installCmd = fmt.Sprintf("winget install --silent %s || choco install -y %s", name, name)
		   } else if isLinux() {
			   // Use apt-get for Linux
			   installCmd = fmt.Sprintf("sudo apt-get update && sudo apt-get install -y %s", name)
		   } else if isMac() {
			   // Use brew for macOS
			   installCmd = fmt.Sprintf("brew install %s", name)
		   }
	   default:
		   fmt.Printf("[Survival] No install logic for tool: %s\n", name)
		   return
	   }
	   if installCmd != "" {
		   fmt.Printf("[Survival] Running install command: %s\n", installCmd)
		   success := runInstallCommandWithRecovery(name, installCmd)
		   if !success {
			   // Try fallback alternatives if available
			   fallback := getFallbackTool(name)
			   if fallback != "" {
				   fmt.Printf("[Survival] Attempting fallback install for %s: %s\n", name, fallback)
				   runInstallCommandWithRecovery(fallback, getInstallCmd(fallback))
			   } else {
				   fmt.Printf("[Survival] No fallback available for tool: %s\n", name)
			   }
		   }
	   }
	}

	func runInstallCommandWithRecovery(name, cmd string) bool {
	   out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	   if err != nil {
		   fmt.Printf("[Survival] Install command failed for %s: %v\nOutput: %s\n", name, err, string(out))
		   // Escalate feedback and update improvement plan
		   escalateRecoveryFailure(name, err)
		   return false
	   }
	   fmt.Printf("[Survival] Install command succeeded for %s. Output: %s\n", name, string(out))
	   return true
	}

	func escalateRecoveryFailure(name string, err error) {
	   // Log, escalate feedback, and update improvement plan
	   fmt.Printf("[Survival] CRITICAL: Failed to recover tool: %s, error: %v\n", name, err)
	   // Optionally: trigger self-repair event, update plan, notify agent
	   // This function can be extended to notify other modules or trigger emergency routines
	}

	func getFallbackTool(name string) string {
	   // Map tool to fallback alternatives
	   switch name {
	   case "curl":
		   return "wget"
	   case "wget":
		   return "curl"
	   case "gh":
		   return "git"
	   default:
		   return ""
	   }
	}

	func getInstallCmd(name string) string {
	   if isWindows() {
		   return fmt.Sprintf("winget install --silent %s || choco install -y %s", name, name)
	   } else if isLinux() {
		   return fmt.Sprintf("sudo apt-get update && sudo apt-get install -y %s", name)
	   } else if isMac() {
		   return fmt.Sprintf("brew install %s", name)
	   }
	   return ""
	}
	}

	func runInstallCommand(cmd string) {
	   // Actually run the install command
	   out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	   if err != nil {
		   fmt.Printf("[Survival] Install command failed: %v\nOutput: %s\n", err, string(out))
	   } else {
		   fmt.Printf("[Survival] Install command succeeded. Output: %s\n", string(out))
	   }
	}

	func isWindows() bool {
	   return runtime.GOOS == "windows"
	}

	func isLinux() bool {
	   return runtime.GOOS == "linux"
	}

	func isMac() bool {
	   return runtime.GOOS == "darwin"
	}
}
   return &Brain{
	   ShortTerm:    shortTerm,
	   LongTerm:     longTerm,
	   Timeline:     make([]Event, 0, 1024),
	   SelfReflect:  NewSelfReflectionModule(),
	   reflectEvery: 10, // Reflect every 10 events by default
	   reflectOn:    map[string]bool{"inbound_message": true, "user_feedback": true, "self_repair": true},
	   ReportFunc:   nil,
	   modules:      make(map[string]any),
	   Security:     NewSecurityModule(),
   }
}

// LogEvent appends an event to the timeline and triggers self-reflection if needed
func (b *Brain) LogEvent(eventType string, data any) error {
   b.mu.Lock()
   defer b.mu.Unlock()
   b.Timeline = append(b.Timeline, Event{
	   Timestamp: time.Now(),
	   Type:      eventType,
	   Data:      data,
   })
   // Record event in self-reflection module
   outcome := "neutral"
   if eventType == "user_feedback" {
	   if str, ok := data.(string); ok && str == "success" {
		   outcome = "success"
	   } else if str, ok := data.(string); ok && str == "failure" {
		   outcome = "failure"
	   }
   }
   if eventType == "self_repair" {
	   outcome = "critical_failure"
   }
   if b.SelfReflect != nil {
	   b.SelfReflect.RecordEvent(eventType, outcome, []string{"brain", eventType})
   }
   b.eventCount++
   // Adaptive: reflect on certain event types or every N events
   if b.reflectOn[eventType] || b.eventCount >= b.reflectEvery {
	   go b.reflectAndReport()
	   b.eventCount = 0
   }
   return nil
}

// SetReflectionFrequency allows user/agent to tune learning frequency
func (b *Brain) SetReflectionFrequency(n int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.reflectEvery = n
}

// RegisterModule allows dynamic extension of the brain with new modules
func (b *Brain) RegisterModule(name string, module any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.modules[name] = module
}

// GetModule retrieves a registered module by name
func (b *Brain) GetModule(name string) (any, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	m, ok := b.modules[name]
	return m, ok
}

// SetReflectionEvents allows user/agent to tune which events trigger reflection
func (b *Brain) SetReflectionEvents(events []string) {
   b.mu.Lock()
   defer b.mu.Unlock()
   b.reflectOn = make(map[string]bool)
   for _, e := range events {
	   b.reflectOn[e] = true
   }
}

// SetReportFunc allows user/agent to set a callback for reflection reports
func (b *Brain) SetReportFunc(f func(summary string)) {
   b.mu.Lock()
   defer b.mu.Unlock()
   b.ReportFunc = f
}

// reflectAndReport runs self-reflection and reports summary if callback is set
func (b *Brain) reflectAndReport() {
	   // Survival: check for required tools and attempt self-repair if missing
	requiredTools := []string{"tmux", "summarize", "gh", "devmem", "curl", "wget", "git"}
	   missingTools := []string{}
	   for _, tool := range requiredTools {
		   if !isToolAvailable(tool) {
			   missingTools = append(missingTools, tool)
			   autoInstallTool(tool)
		   }
	   }
	   if len(missingTools) > 0 {
		   b.LogEvent("self_repair", fmt.Sprintf("missing_tools:%v", missingTools))
		   b.SelfReflect.improvementPlan.Steps = append(b.SelfReflect.improvementPlan.Steps, fmt.Sprintf("Recovered missing tools: %v", missingTools))
		   b.SelfReflect.improvementPlan.Goal = "Survive and restore toolchain"
	   }
   if b.SelfReflect != nil {
	   b.SelfReflect.Analyze()
	   // Prioritize survival and optimal improvement
	   var autoFeedback string
	   steps := b.SelfReflect.improvementPlan.Steps
	   goal := b.SelfReflect.improvementPlan.Goal
	   // Survival: check memory modules and attempt self-repair if missing
	   if b.ShortTerm == nil || b.LongTerm == nil {
		   autoFeedback = "critical_failure"
		   b.LogEvent("self_repair", "memory_module_missing")
		   if b.ShortTerm == nil {
			   b.ShortTerm = NewInMemoryMemory()
		   }
		   if b.LongTerm == nil {
			   b.LongTerm = NewInMemoryMemory()
		   }
		   b.SelfReflect.improvementPlan.Steps = append(b.SelfReflect.improvementPlan.Steps, "Recovered missing memory modules")
		   b.SelfReflect.improvementPlan.Goal = "Survive and restore memory"
	   } else if len(steps) == 0 {
		   autoFeedback = "critical_failure"
		   // Emergency: no improvement steps, must act to survive
		   b.SelfReflect.improvementPlan.Steps = append(b.SelfReflect.improvementPlan.Steps, "Initiate emergency recovery", "Seek new strategies", "Increase learning rate")
		   b.SelfReflect.improvementPlan.Goal = "Survive and adapt immediately"
	   } else if goal == "Reduce mistakes and learn from them" {
		   autoFeedback = "urgent_improvement_needed"
		   // Mistakes outweigh successes, act assertively
		   b.SelfReflect.improvementPlan.Steps = append(b.SelfReflect.improvementPlan.Steps, "Double-check decisions", "Request external review if possible")
	   } else if goal == "Continuous improvement" {
		   autoFeedback = "optimal"
		   // System is thriving, reinforce strengths
		   b.SelfReflect.improvementPlan.Steps = append(b.SelfReflect.improvementPlan.Steps, "Maintain resilience", "Explore new opportunities")
	   } else {
		   autoFeedback = "adaptive"
		   // Default: keep adapting
		   b.SelfReflect.improvementPlan.Steps = append(b.SelfReflect.improvementPlan.Steps, "Monitor progress", "Refine strategies")
	   }
	   b.LogEvent("self_feedback", autoFeedback)
	   if b.ReportFunc != nil {
		   summary := "[Brain] Self-reflection complete at " + time.Now().Format(time.RFC3339) + ". Improvement plan: " + b.SelfReflect.improvementPlan.Goal + ". Steps: " + joinSteps(b.SelfReflect.improvementPlan.Steps) + ". Feedback: " + autoFeedback
		   b.ReportFunc(summary)
	   }
   }
}

func joinSteps(steps []string) string {
   if len(steps) == 0 {
	   return "None"
   }
   result := ""
   for i, s := range steps {
	   if i > 0 {
		   result += ", "
	   }
	   result += s
   }
   return result
}

// Summarize compresses old events into a summary (stub)
func (b *Brain) Summarize() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.Timeline) == 0 {
		return nil
	}
	summary := "Summary of recent events:\n"
	for i, ev := range b.Timeline {
		if i >= 20 {
			break
		}
		summary += ev.Timestamp.Format(time.RFC3339) + ": " + ev.Type + "\n"
	}
	key := "summary:" + time.Now().Format("20060102T150405")
	err := b.LongTerm.Remember(key, summary)
	if err != nil && b.ReportFunc != nil {
		b.ReportFunc("[Brain] Error saving summary: " + err.Error())
	}
	if b.ReportFunc != nil {
		b.ReportFunc(summary)
	}
	b.Timeline = b.Timeline[:0]
	return err
}
