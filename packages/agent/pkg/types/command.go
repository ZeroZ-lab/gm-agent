package types

// CommandDeps dependencies for command execution
// Note: Interfaces for services (LLM, Tools, Store) should be defined in valid packages
// to avoid cyclic dependencies. Thus CommandDeps might use `any` or defined
// interfaces if they can be placed in `types` or a separate `interfaces` package.
// However, the doc says `types` is shared.
// To implement `Execute` method on Command struct in `pkg/types`, `CommandDeps` needs to import `LLMGateway` etc.
// But `LLMGateway` (pkg/llm) likely imports `pkg/types` for `Message` struct.
// This creates a CYCLE: types -> llm -> types.
//
// SOLUTION:
// The `Execute` method logic should NOT be in `pkg/types`.
// `pkg/types` should only hold the data structures (Data Transfer Objects).
// The logic (Dispatcher/Executor) should be in `pkg/runtime` or specific handlers.
// The `data-model.md` showed `Execute` on the struct, which works in a comprehensive doc but fails in Go package structure due to cycles.
// We will define the Command structs here, but the execution logic will move to `runtime`'s Dispatcher.
// OR we define minimal interfaces here in `types` that other packages implement.
// Given Go Best Practices ("Accept Interfaces"), we can define interfaces here IF they are small.
// But LLMGateway is complex.
//
// Decision: Remove `Execute` method from `pkg/types` structs.
// These structs will be pure data containers. The `Execute` logic belongs to the Dispatcher in `pkg/runtime`.
// We will keep the `Command` interface for identification.

type Command interface {
	CommandID() string
	CommandType() string
}

type BaseCommand struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

func (c *BaseCommand) CommandID() string   { return c.ID }
func (c *BaseCommand) CommandType() string { return c.Type }

func NewBaseCommand(cmdType string) BaseCommand {
	return BaseCommand{
		ID:   GenerateCommandID(),
		Type: cmdType,
	}
}

// CallLLMCommand
type CallLLMCommand struct {
	BaseCommand
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools,omitempty"`
}

// CallToolCommand
type CallToolCommand struct {
	BaseCommand
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolName   string         `json:"tool_name"`
	Arguments  map[string]any `json:"arguments"`
}

// ApplyPatchCommand
type ApplyPatchCommand struct {
	BaseCommand
	FilePath string `json:"file_path"`
	Diff     string `json:"diff"`
	DryRun   bool   `json:"dry_run"`
}

// SaveCheckpointCommand
type SaveCheckpointCommand struct {
	BaseCommand
}

// RestoreBackupCommand (mentioned in runtime.md but missing in data-model.md, adding for completeness)
type RestoreBackupCommand struct {
	BaseCommand
	PatchID string `json:"patch_id"`
}
