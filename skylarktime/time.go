// Package time provides time.Time wrappers for the skylark embedded language.
// See https://github.com/google/skylark/issues/17.
package skylarktime

import (
	"errors"
	"sync"
	"time"

	"github.com/google/skylark"
	"github.com/google/skylark/skylarkstruct"
	"github.com/google/skylark/syntax"
)

/*
TODO: provide Skylark module documentation.

module time

functions
        duration(string) duration                               # parse a duration
        location(string) location                               # parse a location
        time(string, format=..., location=...) time             # parse a time
        now() time # implementations would be able to make this a constant
        zero time # a constant

type duration
operators
        duration - time = duration
        duration + time = time
        duration == duration
        duration < duration
fields
        hour int
        minute int
        nanosecond int
        second int

type time
operators
        time == time
        time < time
        time + duration = time
        time - duration = time
        time - time = duration
fields
        year int
        month int
        day int
        hour int
        minute int
        second int
        nanosecond int

TODO:
- unix time_t conversions
- timezone stuff
- strftime formatting
- constructor from 6 components + location

*/

// TODO: the Skylark module system is poor.

var (
	once       sync.Once
	timeModule skylark.StringDict
	timeErr    error
)

// LoadTimeModule loads the time module.
// It is concurrency-safe and idempotent.
func LoadTimeModule() (skylark.StringDict, error) {
	once.Do(func() {
		timeModule = skylark.StringDict{
			"time": skylarkstruct.FromStringDict(
				skylarkstruct.Default,
				skylark.StringDict{
					"now":         skylark.NewBuiltin("error", now),
					"zero":        Time(time.Time{}),
					"duration":    skylark.NewBuiltin("duration", parseDuration),
					"location":    skylark.NewBuiltin("location", parseLocation),
					"time":        skylark.NewBuiltin("time", parseTime),
					"nanosecond":  Duration(time.Nanosecond),
					"microsecond": Duration(time.Microsecond),
					"millisecond": Duration(time.Millisecond),
					"second":      Duration(time.Second),
					"minute":      Duration(time.Minute),
					"hour":        Duration(time.Hour),
				},
			),
		}
	})
	return timeModule, nil
}

// Now is the function called by Skylark's time.now function.
// By default it is Go's time function, but a client
// may install an alternative function that, for example,
// returns a constant to ensure deterministic execution.
//
// TODO(adonovan): make this a thread-local variable.
var Now = time.Now

// now returns the current time.
func now(_ *skylark.Thread, _ *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	if len(args) > 0 || len(kwargs) != 0 {
		return nil, errors.New("now: unexpected arguments")
	}
	return Time(Now()), nil
}

// Delta returns a duration created from kwargs.  Expected "hours", "minutes",
// "seconds", "milliseconds", or "nanoseconds", and assigned an int.
func Delta(_ *skylark.Thread, _ *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	if len(args) != 0 {
		return nil, errors.New("too many args")
	}
	var d time.Duration
	for _, t := range kwargs {
		if len(t) != 2 {
			panic("invalid kwarg")
		}
		s, ok := t[0].(skylark.String)
		if !ok {
			panic("invalid kwarg name")
		}
		i, ok := t[1].(skylark.Int)
		if !ok {
			return nil, errors.New("invalid value for timedelta arg, must be int")
		}
		v, ok := i.Int64()
		if !ok {
			return nil, errors.New("numeric value overflows int64")
		}
		switch s {
		case "hours":
			d += time.Hour * time.Duration(v)
		case "minutes":
			d += time.Minute * time.Duration(v)
		case "seconds":
			d += time.Second * time.Duration(v)
		case "milliseconds":
			d += time.Millisecond * time.Duration(v)
		case "nanoseconds":
			d += time.Nanosecond * time.Duration(v)
		default:
			return nil, errors.New("invalid duration unit: " + string(s))
		}
	}
	return Duration(d), nil
}

// Time is the type of a Skylark time.Time.
type Time time.Time

var (
	_ skylark.Value      = Time{}
	_ skylark.Comparable = Time{}
	_ skylark.HasAttrs   = Time{}
	_ skylark.HasBinary  = Time{}
)

func (t Time) String() string        { return time.Time(t).String() }
func (t Time) Type() string          { return "time.time" }
func (t Time) Freeze()               {} // immutable
func (t Time) Truth() skylark.Bool   { return skylark.Bool(!time.Time(t).IsZero()) }
func (t Time) Hash() (uint32, error) { return uint32(time.Time(t).Unix()), nil } // TODO not robust

func (t Time) CompareSameType(op syntax.Token, y_ skylark.Value, depth int) (bool, error) {
	x := time.Time(t)
	y := time.Time(y_.(Time))
	switch op {
	case syntax.EQL:
		return x.Equal(y), nil
	case syntax.NEQ:
		return !x.Equal(y), nil
	case syntax.LE:
		return !y.Before(x), nil
	case syntax.LT:
		return x.Before(y), nil
	case syntax.GE:
		return !y.After(x), nil
	case syntax.GT:
		return x.After(y), nil
	}
	panic(op)
}

func (t Time) AttrNames() []string {
	return []string{"year", "month", "day", "hour", "minute", "second", "nanosecond"}
}

func (t Time) Attr(name string) (skylark.Value, error) {
	x := time.Time(t)
	switch name {
	case "year":
		return skylark.MakeInt(x.Year()), nil
	case "month":
		return skylark.String(x.Month().String()), nil
	case "day":
		return skylark.MakeInt(x.Day()), nil
	case "hour":
		return skylark.MakeInt(x.Hour()), nil
	case "minute":
		return skylark.MakeInt(x.Minute()), nil
	case "second":
		return skylark.MakeInt(x.Second()), nil
	case "nanosecond":
		return skylark.MakeInt(x.Nanosecond()), nil
	}
	return nil, nil // no such attribute
}

func parseTime(_ *skylark.Thread, b *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var s string
	if err := skylark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &s); err != nil {
		return nil, err
	}
	t, err := time.Parse(time.UnixDate, s) // TODO
	if err != nil {
		return nil, err
	}
	return Time(t), nil
}

func (t Time) Binary(op syntax.Token, y skylark.Value, side skylark.Side) (skylark.Value, error) {
	if side == skylark.Right {
		return nil, nil
	}
	if time2, ok := y.(Time); ok {
		return t.binaryTime(op, time2)
	}
	if d, ok := y.(Duration); ok {
		return t.binaryDuration(op, d)
	}

	return nil, nil
}

func (t Time) binaryTime(op syntax.Token, y Time) (skylark.Value, error) {
	switch op {
	case syntax.MINUS:
		return Duration(time.Time(t).Sub(time.Time(y))), nil
	}
	return nil, nil
}

func (t Time) binaryDuration(op syntax.Token, y Duration) (skylark.Value, error) {
	switch op {
	case syntax.MINUS:
		return Time(time.Time(t).Add(-time.Duration(y))), nil
	case syntax.PLUS:
		return Time(time.Time(t).Add(time.Duration(y))), nil
	}
	return nil, nil
}

type Duration time.Duration

var (
	_ skylark.Value      = Duration(0)
	_ skylark.Comparable = Duration(0)
	_ skylark.HasBinary  = Duration(0)
)

func (d Duration) String() string        { return time.Duration(d).String() }
func (d Duration) Type() string          { return "time.duration" }
func (d Duration) Freeze()               {} // immutable
func (d Duration) Truth() skylark.Bool   { return d != 0 }
func (d Duration) Hash() (uint32, error) { return skylark.MakeInt64(int64(d)).Hash() }

func (x Duration) CompareSameType(op syntax.Token, y_ skylark.Value, depth int) (bool, error) {
	y := y_.(Duration)
	switch op {
	case syntax.EQL:
		return x == y, nil
	case syntax.NEQ:
		return x != y, nil
	case syntax.LE:
		return x <= y, nil
	case syntax.LT:
		return x < y, nil
	case syntax.GE:
		return x >= y, nil
	case syntax.GT:
		return x > y, nil
	}
	panic(op)
}

//         duration + duration = duration
//         duration - duration = duration
//         duration / duration = float
//         duration + time     = time
//         duration * number   = duration
//         number * duration   = duration
//         duration / number   = duration
//
//         time - time = duration
//         time + duration = time
//         time - duration = time
func (x Duration) Binary(op syntax.Token, y_ skylark.Value, side skylark.Side) (skylark.Value, error) {
	if side == skylark.Left {
		// duration op y
		switch y := y_.(type) {
		case Duration:
			// duration + duration
			// duration - duration
			// duration / duration
			switch op {
			case syntax.PLUS:
				return Duration(x + y), nil
			case syntax.MINUS:
				return Duration(x - y), nil
			case syntax.SLASH:
				return skylark.Float(x) / skylark.Float(y), nil
			}
		case Time:
			// duration + time = time
			if op == syntax.PLUS {
				return Time(time.Time(y).Add(time.Duration(x))), nil
			}
		case skylark.Int, skylark.Float:
			// (double x//y not supported)
			if op == syntax.STAR || op == syntax.SLASH {
				return scaleDuration(x, y, op == syntax.SLASH)
			}
		}
	} else {
		// y op duration
		// We need handle only cases not covered by side==Left.
		switch y := y_.(type) {
		case skylark.Int, skylark.Float:
			if op == syntax.STAR {
				// duration * number = duration
				return scaleDuration(x, y, false)
			}
		}
	}

	return nil, nil
}

// duration * k = duration
// k * duration = duration
func scaleDuration(x Duration, y skylark.Value, divide bool) (skylark.Value, error) {
	switch y := y.(type) {
	case skylark.Int:
		// TODO: check for overflow
		// TODO: handle Uint64, bigint.
		if y, ok := y.Int64(); ok {
			if divide {
				return x / Duration(y), nil
			} else {
				return x * Duration(y), nil
			}
		}
	case skylark.Float:
		if divide {
			return Duration(float64(x) / float64(y)), nil
		} else {
			return Duration(float64(x) * float64(y)), nil
		}
	}
	return nil, nil
}

func parseDuration(_ *skylark.Thread, b *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var s string
	if err := skylark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &s); err != nil {
		return nil, err
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return nil, err
	}
	return Duration(d), nil
}

func parseLocation(_ *skylark.Thread, _ *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	return nil, nil
}
