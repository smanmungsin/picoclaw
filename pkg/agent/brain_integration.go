package agent

import (
	"github.com/sipeed/picoclaw/pkg/brain"
)

// AgentBrain wraps the brain for agent use
// This can be extended to expose more modules/APIs as needed
type AgentBrain struct {
	Brain *brain.Brain
}

func NewAgentBrain() *AgentBrain {
   shortTerm := brain.NewInMemoryMemory()
   longTerm, err := brain.NewBadgerMemory("./brain_data")
   if err != nil {
	   fmt.Println("[Brain] Warning: Falling back to in-memory long-term memory:", err)
	   longTerm = shortTerm
   }
   b := brain.NewBrain(shortTerm, longTerm)
   // Wire up reporting to log output
   b.SetReportFunc(func(summary string) {
	   fmt.Println(summary) // Print to console/log for both user and agent
   })

   // Connect conversation module for adaptive learning
   conv := brain.NewConversationModule()
   conv.OnAdd = func(msg string) {
	   defer func() {
		   if r := recover(); r != nil {
			   fmt.Println("[Brain] Conversation callback error:", r)
		   }
	   }()
	   b.LogEvent("conversation_message", msg)
	   // Optionally trigger summarization or adaptation here
	   if len(conv.GetHistory(20))%10 == 0 { // Every 10 messages
		   go b.Summarize()
	   }
   }
   // Optionally expose conv as a field or module
   return &AgentBrain{Brain: b}
}

// Example: Log an event to the brain
defaultAgentBrain *AgentBrain

func GetAgentBrain() *AgentBrain {
	if defaultAgentBrain == nil {
		defaultAgentBrain = NewAgentBrain()
	}
	return defaultAgentBrain
}
