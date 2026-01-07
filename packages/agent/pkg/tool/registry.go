package tool

import (
	"fmt"
	"sync"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

type Registry struct {
	mu    sync.RWMutex
	tools map[string]types.Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]types.Tool),
	}
}

func (r *Registry) Register(tool types.Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool %s already registered", tool.Name)
	}
	r.tools[tool.Name] = tool
	return nil
}

func (r *Registry) Get(name string) (types.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) List() []types.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]types.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}
