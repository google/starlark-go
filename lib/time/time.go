package time

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"
)

// ModuleName defines the expected name for this Module when used in the
// starlark runtime.
const ModuleName = "time"

// Module time is a Starlark module of time-related functions.
var Module = &starlarkstruct.Module{
	Name: ModuleName,
	Members: starlark.StringDict{
		"parse_duration": starlark.NewBuiltin("parse_duration", parseDuration),
		"parse_location": starlark.NewBuiltin("parse_location", parseLocation),
		"now":            starlark.NewBuiltin("now", now),
		"time":           starlark.NewBuiltin("time", newTime),
		"parse_time":     starlark.NewBuiltin("time", parseTime),
		"sleep":          starlark.NewBuiltin("sleep", sleep),
		"from_timestamp": starlark.NewBuiltin("from_timestamp", fromTimestamp),

		"nanosecond":  Duration(time.Nanosecond),
		"microsecond": Duration(time.Microsecond),
		"millisecond": Duration(time.Millisecond),
		"second":      Duration(time.Second),
		"minute":      Duration(time.Minute),
		"hour":        Duration(time.Hour),
	},
}

// LoadModule loads the time module.
// It is concurrency-safe and idempotent.
func LoadModule() (starlark.StringDict, error) {
	return starlark.StringDict{
		ModuleName: Module,
	}, nil
}

// NowFunc is a function that generates the current time. Intentionally exported
// so that it can be overridden, for example by applications that require their
// Starlark scripts to be fully deterministic.
var NowFunc = time.Now

func parseDuration(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var d Duration
	err := starlark.UnpackPositionalArgs("duration", args, kwargs, 1, &d)
	return d, err
}

func parseLocation(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var s string
	if err := starlark.UnpackPositionalArgs("location", args, kwargs, 1, &s); err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(s)
	if err != nil {
		return nil, err
	}

	return starlark.String(loc.String()), nil
}

// SleepFunc is a function that pauses the current goroutine for at least d.
// Intentionally exported so that it can be overridden, for example by
// applications that require their Starlark scripts to be fully deterministic.
var SleepFunc = time.Sleep

func sleep(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var dur Duration
	if err := starlark.UnpackPositionalArgs("sleep", args, kwargs, 1, &dur); err != nil {
		return starlark.None, err
	}

	SleepFunc(time.Duration(dur))
	return starlark.None, nil
}

func parseTime(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		x, location string
		format      = time.RFC3339
	)
	if err := starlark.UnpackArgs("time", args, kwargs, "x", &x, "format?", &format, "location?", &location); err != nil {
		return nil, err
	}

	if location == "" {
		t, err := time.Parse(format, x)
		if err != nil {
			return nil, err
		}
		return Time(t), nil
	}

	loc, err := time.LoadLocation(location)
	if err != nil {
		return nil, err
	}
	t, err := time.ParseInLocation(format, x, loc)
	if err != nil {
		return nil, err
	}
	return Time(t), nil
}

func fromTimestamp(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var x int
	if err := starlark.UnpackPositionalArgs("from_timestamp", args, kwargs, 1, &x); err != nil {
		return nil, err
	}
	return Time(time.Unix(int64(x), 0)), nil
}

func now(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return Time(NowFunc()), nil
}

// Duration is a Starlark representation of a duration.
type Duration time.Duration

// assert at compile time that Duration implements Unpacker.
var _ starlark.Unpacker = (*Duration)(nil)

// Unpack is a custom argument unpacker
func (d *Duration) Unpack(v starlark.Value) error {
	switch x := v.(type) {
	case Duration:
		*d = x
		return nil
	case starlark.Int:
		i, ok := x.Int64()
		if !ok {
			return fmt.Errorf("int value out of range (want signed 64-bit value)")
		}
		*d = Duration(i)
		return nil
	case starlark.String:
		dur, err := time.ParseDuration(string(x))
		if err != nil {
			return err
		}

		*d = Duration(dur)
		return nil
	}

	return fmt.Errorf("got %s, want a duration, string, or int", v.Type())
}

// String implements the Stringer interface.
func (d Duration) String() string { return time.Duration(d).String() }

// Type returns a short string describing the value's type.
func (d Duration) Type() string { return ModuleName + ".duration" }

// Freeze renders Duration immutable. required by starlark.Value interface
// because duration is already immutable this is a no-op.
func (d Duration) Freeze() {}

// Hash returns a function of x such that Equals(x, y) => Hash(x) == Hash(y)
// required by starlark.Value interface.
func (d Duration) Hash() (uint32, error) {
	return uint32(d) ^ uint32(int64(d)>>32), nil
}

// Truth reports whether the duration is non-zero.
func (d Duration) Truth() starlark.Bool { return d != 0 }

// Attr gets a value for a string attribute, implementing dot expression support
// in starklark. required by starlark.HasAttrs interface.
func (d Duration) Attr(name string) (starlark.Value, error) {
	switch name {
	case "hours":
		return starlark.Float(time.Duration(d).Hours()), nil
	case "minutes":
		return starlark.Float(time.Duration(d).Minutes()), nil
	case "seconds":
		return starlark.Float(time.Duration(d).Seconds()), nil
	case "milliseconds":
		return starlark.MakeInt64(time.Duration(d).Milliseconds()), nil
	case "microseconds":
		return starlark.MakeInt64(time.Duration(d).Microseconds()), nil
	case "nanoseconds":
		return starlark.MakeInt64(time.Duration(d).Nanoseconds()), nil
	}
	return nil, fmt.Errorf("unrecognized %s attribute %q", d.Type(), name)
}

// AttrNames lists available dot expression strings. required by
// starlark.HasAttrs interface.
func (d Duration) AttrNames() []string {
	return []string{
		"hours",
		"minutes",
		"seconds",
		"milliseconds",
		"microseconds",
		"nanoseconds",
	}
}

// CompareSameType implements comparison of two Duration values. required by
// starlark.Comparable interface.
func (d Duration) CompareSameType(op syntax.Token, v starlark.Value, depth int) (bool, error) {
	x := time.Duration(d)
	y := time.Duration(v.(Duration))
	switch op {
	case syntax.EQL:
		return x == y, nil
	case syntax.NEQ:
		return x != y, nil
	case syntax.LT:
		return x < y, nil
	case syntax.LE:
		return x <= y, nil
	case syntax.GT:
		return x > y, nil
	case syntax.GE:
		return x >= y, nil
	}
	return false, errors.New("operation not supported")
}

// Binary implements binary operators, which satisfies the starlark.HasBinary
// interface. operators:
//    duration + duration = duration
//    duration + time = time
//    duration - duration = duration
//    duration / duration = float
//    duration / int = duration
//    duration // duration = int
//    duration // int = duration
//    duration * int = duration
func (d Duration) Binary(op syntax.Token, yV starlark.Value, _ starlark.Side) (starlark.Value, error) {
	x := time.Duration(d)

	switch op {
	case syntax.PLUS:
		switch y := yV.(type) {
		case Duration:
			return Duration(x + time.Duration(y)), nil
		case Time:
			return Time(time.Time(y).Add(x)), nil
		}

	case syntax.MINUS:
		switch y := yV.(type) {
		case Duration:
			return Duration(x - time.Duration(y)), nil
		}

	case syntax.SLASH:
		switch y := yV.(type) {
		case Duration:
			if int64(y) == 0 {
				return nil, fmt.Errorf("%s division by zero", d.Type())
			}
			return starlark.Float(float64(x.Nanoseconds()) / float64(time.Duration(y).Nanoseconds())), nil
		case starlark.Int:
			i, ok := y.Int64()
			if !ok {
				return nil, fmt.Errorf("int value out of range (want signed 64-bit value)")
			}
			if int64(i) == 0 {
				return nil, fmt.Errorf("%s division by zero", d.Type())
			}
			return Duration(d / Duration(i)), nil
		}

	case syntax.SLASHSLASH:
		switch y := yV.(type) {
		case Duration:
			if int64(y) == 0 {
				return nil, fmt.Errorf("%s division by zero", d.Type())
			}
			return starlark.MakeInt64(x.Nanoseconds() / time.Duration(y).Nanoseconds()), nil
		case starlark.Int:
			i, ok := yV.(starlark.Int).Int64()
			if !ok {
				return nil, fmt.Errorf("int value out of range (want signed 64-bit value)")
			}
			if int64(i) == 0 {
				return nil, fmt.Errorf("%s division by zero", d.Type())
			}
			return d / Duration(i), nil
		}

	case syntax.STAR:
		switch y := yV.(type) {
		case starlark.Int:
			i, ok := y.Int64()
			if !ok {
				return nil, fmt.Errorf("int value out of range (want signed 64-bit value)")
			}
			return d * Duration(i), nil
		}
	}

	return nil, nil
}

// Time is a starlark representation of a point in time.
type Time time.Time

func newTime(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		year, month, day, hour, min, sec, nsec int
		loc                                    string
	)
	if err := starlark.UnpackArgs("time", args, kwargs, "year?", &year, "month?", &month, "day?", &day, "hour?", &hour, "minute?", &min, "second?", &sec, "nanosecond?", &nsec, "location?", &loc); err != nil {
		return nil, err
	}
	location, err := time.LoadLocation(loc)
	if err != nil {
		return nil, err
	}
	return Time(time.Date(year, time.Month(month), day, hour, min, sec, nsec, location)), nil
}

// String returns the time formatted using the format string
//	"2006-01-02 15:04:05.999999999 -0700 MST"
func (t Time) String() string { return time.Time(t).String() }

// Type returns "time.time".
func (t Time) Type() string { return ModuleName + ".time" }

// Freeze renders time immutable. required by starlark.Value interface
// because Time is already immutable this is a no-op.
func (t Time) Freeze() {}

// Hash returns a function of x such that Equals(x, y) => Hash(x) == Hash(y)
// required by starlark.Value interface.
func (t Time) Hash() (uint32, error) {
	return uint32(time.Time(t).UnixNano()) ^ uint32(int64(time.Time(t).UnixNano())>>32), nil
}

// Truth returns the truth value of an object required by starlark.Value
// interface.
func (t Time) Truth() starlark.Bool { return starlark.Bool(time.Time(t).IsZero()) }

// Attr gets a value for a string attribute, implementing dot expression support
// in starklark. required by starlark.HasAttrs interface.
func (t Time) Attr(name string) (starlark.Value, error) {
	switch name {
	case "year":
		return starlark.MakeInt(time.Time(t).Year()), nil
	case "month":
		return starlark.MakeInt(int(time.Time(t).Month())), nil
	case "day":
		return starlark.MakeInt(time.Time(t).Day()), nil
	case "hour":
		return starlark.MakeInt(time.Time(t).Hour()), nil
	case "minute":
		return starlark.MakeInt(time.Time(t).Minute()), nil
	case "second":
		return starlark.MakeInt(time.Time(t).Second()), nil
	case "nanosecond":
		return starlark.MakeInt(time.Time(t).Nanosecond()), nil
	case "unix":
		return starlark.MakeInt64(time.Time(t).Unix()), nil
	case "unix_nano":
		return starlark.MakeInt64(time.Time(t).UnixNano()), nil
	}
	return builtinAttr(t, name, timeMethods)
}

// AttrNames lists available dot expression strings for time. required by
// starlark.HasAttrs interface.
func (t Time) AttrNames() []string {
	return append(builtinAttrNames(timeMethods),
		"year",
		"month",
		"day",
		"hour",
		"minute",
		"second",
		"nanosecond",
		"unix",
		"unix_nano",
	)
}

// CompareSameType implements comparison of two Time values. required by
// starlark.Comparable interface.
func (t Time) CompareSameType(op syntax.Token, yV starlark.Value, depth int) (bool, error) {
	x := time.Time(t)
	y := time.Time(yV.(Time))
	cp := 0
	if x.Before(y) {
		cp = -1
	} else if x.After(y) {
		cp = 1
	}
	return threeway(op, cp), nil
}

// Binary implements binary operators, which satisfies the starlark.HasBinary
// interface
//    time + duration = time
//    time - duration = time
//    time - time = duration
func (t Time) Binary(op syntax.Token, yV starlark.Value, side starlark.Side) (starlark.Value, error) {
	x := time.Time(t)

	switch op {
	case syntax.PLUS:
		switch y := yV.(type) {
		case Duration:
			return Time(x.Add(time.Duration(y))), nil
		}
	case syntax.MINUS:
		switch y := yV.(type) {
		case Duration:
			return Time(x.Add(time.Duration(-y))), nil
		case Time:
			// time - time = duration
			if side == starlark.Left {
				return Duration(x.Sub(time.Time(y))), nil
			} else {
				return Duration(time.Time(y).Sub(x)), nil
			}
		}
	}

	return nil, nil
}

var timeMethods = map[string]builtinMethod{
	"in_location": timeIn,
	"format":      timeFormat,
}

func timeFormat(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var x string
	if err := starlark.UnpackPositionalArgs("format", args, kwargs, 1, &x); err != nil {
		return nil, err
	}

	recv := time.Time(recV.(Time))
	return starlark.String(recv.Format(x)), nil
}

func timeIn(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var x string
	if err := starlark.UnpackPositionalArgs("in_location", args, kwargs, 1, &x); err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(x)
	if err != nil {
		return nil, err
	}

	recv := time.Time(recV.(Time))
	return Time(recv.In(loc)), nil
}

type builtinMethod func(fnname string, recv starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error)

func builtinAttr(recv starlark.Value, name string, methods map[string]builtinMethod) (starlark.Value, error) {
	method := methods[name]
	if method == nil {
		return nil, nil // no such method
	}

	// Allocate a closure over 'method'.
	impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return method(b.Name(), b.Receiver(), args, kwargs)
	}
	return starlark.NewBuiltin(name, impl).BindReceiver(recv), nil
}

func builtinAttrNames(methods map[string]builtinMethod) []string {
	names := make([]string, 0, len(methods))
	for name := range methods {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// threeway interprets a three-way comparison value cmp (-1, 0, +1)
// as a boolean comparison (e.g. x < y).
func threeway(op syntax.Token, cmp int) bool {
	switch op {
	case syntax.EQL:
		return cmp == 0
	case syntax.NEQ:
		return cmp != 0
	case syntax.LE:
		return cmp <= 0
	case syntax.LT:
		return cmp < 0
	case syntax.GE:
		return cmp >= 0
	case syntax.GT:
		return cmp > 0
	}
	panic(op)
}
