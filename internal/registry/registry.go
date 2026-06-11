// Package registry holds the set of available tasks in a stable order and is
// the single source of truth shared by the CLI runner and the MCP server.
package registry

import "github.com/JadoJodo/rundown/internal/task"

// Registry stores registered tasks and preserves registration order.
type Registry struct {
	order []string
	tasks map[string]task.Task
}

// New returns an empty Registry.
func New() *Registry {
	return &Registry{tasks: map[string]task.Task{}}
}

// Register adds t. A later registration with the same name replaces the task
// but keeps its original position in the ordering.
func (r *Registry) Register(t task.Task) {
	name := t.Name()
	if _, exists := r.tasks[name]; !exists {
		r.order = append(r.order, name)
	}
	r.tasks[name] = t
}

// Get returns the task registered under name.
func (r *Registry) Get(name string) (task.Task, bool) {
	t, ok := r.tasks[name]
	return t, ok
}

// All returns the tasks in registration order.
func (r *Registry) All() []task.Task {
	out := make([]task.Task, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.tasks[name])
	}
	return out
}

// Names returns the task names in registration order.
func (r *Registry) Names() []string {
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}
