// Copyright 2020 Honda Research Institute Europe GmbH. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package starlarktime

import (
	"fmt"
	"time"

	"go.starlark.net/starlark"
)

var starlarkTimeBuiltins = map[string]*starlark.Builtin{
	"timestamp": starlark.NewBuiltin("timestamp", getTimestamp),
	"weekday": starlark.NewBuiltin("weekday", getWeekday),
}

func getTimestamp(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 0); err != nil {
		return starlark.None, err
	}

	t, ok := b.Receiver().(*StarlarkTime)
	if ! ok {
		return starlark.None, fmt.Errorf("%v is not a time", b.Name())
	}

	return starlark.MakeInt64(t.Time.UnixNano()), nil
}

func getWeekday(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 0); err != nil {
		return starlark.None, err
	}

	t, ok := b.Receiver().(*StarlarkTime)
	if ! ok {
		return starlark.None, fmt.Errorf("%v is not a time", b.Name())
	}

	weekday := t.Time.Weekday()
	if weekday == time.Sunday {
		return starlark.MakeInt(7), nil
	}

	return starlark.MakeInt(int(weekday)), nil
}
