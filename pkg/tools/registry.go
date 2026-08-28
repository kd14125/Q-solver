package tools

import "sync"

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewRegistry() *Registry           { return &Registry{tools: make(map[string]Tool)} }
func (r *Registry) Register(tool Tool) { r.mu.Lock(); defer r.mu.Unlock(); r.tools[tool.Name()] = tool }
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}
func (r *Registry) Has(name string) bool { _, ok := r.Get(name); return ok }
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}
func (r *Registry) Execute(ctx *ToolContext, toolID, name string) *ToolResult {
	tool, ok := r.Get(name)
	if !ok {
		return &ToolResult{Text: "未知工具: " + name}
	}
	return tool.Execute(ctx, toolID)
}

var DefaultRegistry = NewRegistry()

func Register(tool Tool) { DefaultRegistry.Register(tool) }
func Execute(ctx *ToolContext, toolID, name string) *ToolResult {
	return DefaultRegistry.Execute(ctx, toolID, name)
}
