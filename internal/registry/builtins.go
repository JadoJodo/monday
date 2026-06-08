package registry

import (
	"github.com/JadoJodo/monday/internal/tasks/custom"
	"github.com/JadoJodo/monday/internal/tasks/mas"
	"github.com/JadoJodo/monday/internal/tasks/npm"
	"github.com/JadoJodo/monday/internal/tasks/softwareupdate"
)

// Default returns a Registry populated with the built-in tasks in their
// canonical execution order.
func Default() *Registry {
	r := New()
	r.Register(softwareupdate.New())
	r.Register(mas.New())
	r.Register(npm.New())
	r.Register(custom.New())
	return r
}
