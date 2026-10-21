// Copyright 2026 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package starlark

import (
	"runtime/pprof"

	"go.starlark.net/internal/compile"
)

func init() {
	fr := pprof.Var[*frame]()

	pprof.RegisterVirtualFrame(
		// CallInternal provides a virtual frame
		// via the 'fr' local variable.
		(*Function).CallInternal,

		// To obtain the current frame's "code" component,
		// evaluate fr.callable.(*Function).funcode.
		fr.IfaceField(func(fr *frame) *Callable { return &fr.callable }).
			TypeAssert[*Function]().
			PtrField(func(fn *Function) **compile.Funcode { return &fn.funcode }),

		// To obtain the current frame's "pc" component,
		// evaluate fr.pc (loaded as uint32).
		fr.ScalarField(func(fr *frame) *uint32 { return &fr.pc }),

		// To convert the obtained (code, pc) pair to a (symbol, file, line)
		// triple, the pprof encoder executes this code.
		func(fn *compile.Funcode, pc uint32) (fnname, file string, line int) {
			pos := fn.Position(pc)
			return fn.Name, pos.Filename(), int(pos.Line)
		},
	)
}
