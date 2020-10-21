package starlark

import "fmt"

type Module struct {
	Name    string
	Members StringDict
}

var _ HasAttrs = (*Module)(nil)

func (m *Module) Attr(name string) (Value, error) { return m.Members[name], nil }
func (m *Module) AttrNames() []string             { return m.Members.Keys() }
func (m *Module) Freeze()                         { m.Members.Freeze() }
func (m *Module) Hash() (uint32, error)           { return 0, fmt.Errorf("unhashable: %s", m.Type()) }
func (m *Module) String() string                  { return fmt.Sprintf("<module %q>", m.Name) }
func (m *Module) Truth() Bool                     { return true }
func (m *Module) Type() string                    { return "module" }

// MakeModule may be used as the implementation of a Starlark built-in
// function, module(name, **kwargs). It returns a new module with the
// specified name and members.
func MakeModule(thread *Thread, b *Builtin, args Tuple, kwargs []Tuple) (Value, error) {
	var name string
	if err := UnpackPositionalArgs(b.Name(), args, nil, 1, &name); err != nil {
		return nil, err
	}
	members := make(StringDict, len(kwargs))
	for _, kwarg := range kwargs {
		k := string(kwarg[0].(String))
		members[k] = kwarg[1]
	}
	return &Module{name, members}, nil
}
