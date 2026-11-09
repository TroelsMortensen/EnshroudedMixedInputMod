package mod

import (
	"github.com/trmo/EnshroudedMixInputMod/mod/inputtrace"
	"github.com/trmo/EnshroudedMixInputMod/mod/mixedinput"
)

type Mod interface {
	Name() string
	Install() error
	Installed() (bool, error)
	Remove() error
}

var mods = []Mod{}

func All() []Mod {
	if len(mods) == 0 {
		mods = []Mod{
			mixedinput.New(),
			inputtrace.New(), // temporary: remove after Phase 2.1
		}
	}

	return mods
}

func ModByType[T any]() T {
	for _, mod := range All() {
		if m, ok := mod.(T); ok {
			return m
		}
	}

	var zero T

	return zero
}
