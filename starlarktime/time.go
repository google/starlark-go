// Copyright 2020 Honda Research Institute Europe GmbH. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package starlarktime

import (
	"errors"
	"sort"
	"time"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type StarlarkTime struct {
	Time   time.Time
	frozen bool
}

// >>> Implementation of starlark Value interface
func (t *StarlarkTime) String() string {
	return t.Time.String()
}

func (t *StarlarkTime) Type() string {
	return "time"
}

func (t *StarlarkTime) Freeze() {
	t.frozen = true
}

func (t *StarlarkTime) Truth() starlark.Bool {
	return true
}

func (t *StarlarkTime) Hash() (uint32, error) {
	return 0, errors.New("not hashable")
}

// <<< Implementation of starlark Value interface

// >>> Implementation of starlark.Comparable interface.
func (t *StarlarkTime) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	a := t.Time
	b := y.(*StarlarkTime).Time

	switch op {
	case syntax.EQL:
		return a.Equal(b), nil
	case syntax.NEQ:
		return !a.Equal(b), nil
	case syntax.LT:
		return a.Before(b), nil
	case syntax.LE:
		return !a.After(b), nil
	case syntax.GT:
		return a.After(b), nil
	case syntax.GE:
		return !a.Before(b), nil
	}

	return false, errors.New("operation not supported")
}

// <<< Implementation of starlark.Comparable interface.

// >>> Implementation of starlark.HasAttrs interface.
func (t *StarlarkTime) AttrNames() []string {
	attrs := []string{"year", "month", "day", "hour", "minute", "second", "microsecond"}
	for name := range starlarkTimeBuiltins {
		attrs = append(attrs, name)
	}
	sort.Strings(attrs)

	return attrs
}

func (t *StarlarkTime) Attr(name string) (starlark.Value, error) {
	switch name {
	case "year":
		return starlark.MakeInt(t.Time.Year()), nil
	case "month":
		return starlark.MakeInt(int(t.Time.Month())), nil
	case "day":
		return starlark.MakeInt(t.Time.Day()), nil
	case "hour":
		return starlark.MakeInt(t.Time.Hour()), nil
	case "minute":
		return starlark.MakeInt(t.Time.Minute()), nil
	case "second":
		return starlark.MakeInt(t.Time.Second()), nil
	case "microsecond":
		return starlark.Float(float64(t.Time.Nanosecond()) / 1e3), nil
	default:
		if builtin, ok := starlarkTimeBuiltins[name]; ok {
			return builtin.BindReceiver(t), nil
		}
		// Returning nil, nil indicates "no such field or method"
		return nil, nil
	}
}

// <<< Implementation of starlark.HasAttrs interface.
