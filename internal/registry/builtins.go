package registry

import (
	"github.com/JadoJodo/rundown/internal/tasks/brew"
	"github.com/JadoJodo/rundown/internal/tasks/cleanup"
	"github.com/JadoJodo/rundown/internal/tasks/custom"
	"github.com/JadoJodo/rundown/internal/tasks/health"
	"github.com/JadoJodo/rundown/internal/tasks/mas"
	"github.com/JadoJodo/rundown/internal/tasks/mise"
	"github.com/JadoJodo/rundown/internal/tasks/npm"
	"github.com/JadoJodo/rundown/internal/tasks/pipx"
	"github.com/JadoJodo/rundown/internal/tasks/rustup"
	"github.com/JadoJodo/rundown/internal/tasks/softwareupdate"
)

// Default returns a Registry populated with the built-in tasks in their
// canonical execution order: package upgrades, then user scripts, then
// read-only reports.
func Default() *Registry {
	r := New()
	r.Register(softwareupdate.New())
	r.Register(mas.New())
	r.Register(brew.New())
	r.Register(npm.New())
	r.Register(pipx.New())
	r.Register(rustup.New())
	r.Register(mise.New())
	r.Register(custom.New())
	r.Register(cleanup.New())
	r.Register(health.New())
	return r
}
