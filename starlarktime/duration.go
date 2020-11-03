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

type StarlarkDuration struct {
	Duration time.Duration
	frozen   bool
}

// >>> Implementation of starlark Value interface
func (t *StarlarkDuration) String() string {
	return t.Duration.String()
}

func (t *StarlarkDuration) Type() string {
	return "Duration"
}

func (t *StarlarkDuration) Freeze() {
	t.frozen = true
}

func (t *StarlarkDuration) Truth() starlark.Bool {
	return t.Duration != 0
}

func (t *StarlarkDuration) Hash() (uint32, error) {
	return 0, errors.New("not hashable")
}

// <<< Implementation of starlark Value interface

// >>> Implementation of starlark.HasAttrs interface.
func (t *StarlarkDuration) AttrNames() []string {
	attrs := []string{"hours", "minutes", "seconds", "milliseconds", "microseconds", "nanoseconds"}
	sort.Strings(attrs)

	return attrs
}

func (t *StarlarkDuration) Attr(name string) (starlark.Value, error) {
	switch name {
	case "hours":
		return starlark.Float(t.Duration.Hours()), nil
	case "minutes":
		return starlark.Float(t.Duration.Minutes()), nil
	case "seconds":
		return starlark.Float(t.Duration.Seconds()), nil
	case "milliseconds":
		return starlark.Float(t.Duration.Milliseconds()), nil
	case "microseconds":
		return starlark.Float(t.Duration.Microseconds()), nil
	case "nanoseconds":
		return starlark.Float(t.Duration.Nanoseconds()), nil
	default:
		// Returning nil, nil indicates "no such field or method"
		return nil, nil
	}
}

// <<< Implementation of starlark.HasAttrs interface.

// >>> Implementation of starlark.Comparable interface.
func (t *StarlarkDuration) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	a := t.Duration
	b := y.(*StarlarkDuration).Duration

	switch op {
	case syntax.EQL:
		return a == b, nil
	case syntax.NEQ:
		return a != b, nil
	case syntax.LT:
		return a < b, nil
	case syntax.LE:
		return a <= b, nil
	case syntax.GT:
		return a > b, nil
	case syntax.GE:
		return a >= b, nil
	}

	return false, errors.New("operation not supported")
}

// <<< Implementation of starlark.Comparable interface.

// >>> Implementation of starlark.HasBinary interface.
func (t *StarlarkDuration) Binary(op syntax.Token, sy starlark.Value, side starlark.Side) (starlark.Value, error) {
	// We always expect left-hand operations in the form x = a + b.
	if side != starlark.Left {
		return nil, nil
	}

	switch y := sy.(type) {
	case *StarlarkTime:
		switch op {
		case syntax.PLUS:
			return &StarlarkTime{Time: y.Time.Add(t.Duration)}, nil
		default:
			return nil, nil
		}
	case *StarlarkDuration:
		switch op {
		case syntax.PLUS:
			return &StarlarkDuration{Duration: t.Duration + y.Duration}, nil
		case syntax.MINUS:
			return &StarlarkDuration{Duration: t.Duration - y.Duration}, nil
		case syntax.SLASH:
			return starlark.Float(float64(t.Duration) / float64(y.Duration)), nil
		default:
			return nil, nil
		}
	case starlark.Int:
		s, ok := y.Int64()
		if !ok {
			return nil, errors.New("cannot convert operant to int64")
		}
		switch op {
		case syntax.PLUS:
			return &StarlarkDuration{Duration: t.Duration + time.Duration(s)*time.Second}, nil
		case syntax.MINUS:
			return &StarlarkDuration{Duration: t.Duration - time.Duration(s)*time.Second}, nil
		case syntax.STAR:
			return &StarlarkDuration{Duration: time.Duration(t.Duration.Nanoseconds() * s)}, nil
		case syntax.SLASH:
			return starlark.Float(t.Duration.Seconds() / float64(s)), nil
		default:
			return nil, nil
		}
	case starlark.Float:
		switch op {
		case syntax.PLUS:
			return &StarlarkDuration{Duration: t.Duration + time.Duration(float64(y*1e9))}, nil
		case syntax.MINUS:
			return &StarlarkDuration{Duration: t.Duration - time.Duration(float64(y*1e9))}, nil
		case syntax.STAR:
			return &StarlarkDuration{Duration: time.Duration(float64(t.Duration.Nanoseconds()) * float64(y))}, nil
		case syntax.SLASH:
			return starlark.Float(t.Duration.Seconds() / float64(y)), nil
		default:
			return nil, nil
		}
	}
	return nil, nil
}

// <<< Implementation of starlark.HasBinary interface.
