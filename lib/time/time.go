package time

import (
	"fmt"
	"sort"
	"strconv"
	gotime "time"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"
)

// ModuleName defines the expected name for this Module when used in the
// starlark runtime
const ModuleName = "time"

// Module time is a Starlark module of time-related functions.
var Module = &starlarkstruct.Module{
	Name: ModuleName,
	Members: starlark.StringDict{
		"duration":      starlark.NewBuiltin("duration", duration),
		"location":      starlark.NewBuiltin("location", location),
		"now":           starlark.NewBuiltin("now", now),
		"struct":        starlark.NewBuiltin("struct", starlarkstruct.Make),
		"time":          starlark.NewBuiltin("time", time),
		"sleep":         starlark.NewBuiltin("sleep", sleep),
		"fromtimestamp": starlark.NewBuiltin("fromtimestamp", fromtimestamp),

		"zero": Time{},

		"nanosecond":  durInt(gotime.Nanosecond),
		"microsecond": durInt(gotime.Microsecond),
		"millisecond": durInt(gotime.Millisecond),
		"second":      durInt(gotime.Second),
		"minute":      durInt(gotime.Minute),
		"hour":        durInt(gotime.Hour),
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
// so that it can be overridden
var NowFunc = func() gotime.Time { return gotime.Now() }

func duration(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var x starlark.String
	if err := starlark.UnpackArgs("duration", args, kwargs, "x", &x); err != nil {
		return nil, err
	}

	d, err := gotime.ParseDuration(string(x))
	if err != nil {
		return nil, err
	}

	return Duration(d), nil
}

func durInt(d gotime.Duration) starlark.Int {
	return starlark.MakeInt64(int64(d))
}

func location(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var x starlark.String
	if err := starlark.UnpackArgs("location", args, kwargs, "x", &x); err != nil {
		return nil, err
	}
	loc, err := gotime.LoadLocation(string(x))
	if err != nil {
		return nil, err
	}

	return starlark.String(loc.String()), nil
}

func sleep(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var x starlark.Int
	if err := starlark.UnpackArgs("sleep", args, kwargs, "x", &x); err != nil {
		return starlark.None, err
	}
	dur, ok := x.Int64()
	if !ok {
		return starlark.None, fmt.Errorf("invalid sleep value")
	}

	gotime.Sleep(gotime.Duration(dur))
	return starlark.None, nil
}

func time(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		x, location starlark.String
		format      = starlark.String(gotime.RFC3339)
	)
	if err := starlark.UnpackArgs("time", args, kwargs, "x", &x, "format?", &format, "location", &location); err != nil {
		return nil, err
	}

	if location == "" {
		t, err := gotime.Parse(format.String(), x.String())
		if err != nil {
			return nil, err
		}
		return Time(t), nil
	}

	loc, err := gotime.LoadLocation(location.String())
	if err != nil {
		return nil, err
	}
	t, err := gotime.ParseInLocation(format.String(), x.String(), loc)
	if err != nil {
		return nil, err
	}
	return Time(t), nil
}

func fromtimestamp(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		x starlark.Int
	)
	if err := starlark.UnpackArgs("time", args, kwargs, "x", &x); err != nil {
		return nil, err
	}

	i, err := strconv.ParseInt(x.String(), 10, 64)
	if err != nil {
		return nil, err
	}

	t := gotime.Unix(i, 0)

	return Time(t), nil
}

func now(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return Time(NowFunc()), nil
}

// Duration is a starlark representation of a duration
type Duration gotime.Duration

// String implements the Stringer interface
func (d Duration) String() string { return gotime.Duration(d).String() }

// Type returns a short string describing the value's type.
func (d Duration) Type() string { return "duration" }

// Freeze renders Duration immutable. required by starlark.Value interface
// because duration is already immutable this is a no-op
func (d Duration) Freeze() {}

// Hash returns a function of x such that Equals(x, y) => Hash(x) == Hash(y)
// required by starlark.Value interface
func (d Duration) Hash() (uint32, error) { return hashString(d.String()), nil }

// Truth returns the truth value of an object required by starlark.Value interface
func (d Duration) Truth() starlark.Bool { return d > 0 }

// Attr gets a value for a string attribute, implementing dot expression support in starklark. required by starlark.HasAttrs interface
func (d Duration) Attr(name string) (starlark.Value, error) {
	return builtinAttr(d, name, durationMethods)
}

// AttrNames lists available dot expression strings. required by starlark.HasAttrs interface
func (d Duration) AttrNames() []string { return builtinAttrNames(durationMethods) }

// Binary implements binary operators, which satisfies the starlark.HasBinary interface
func (d Duration) Binary(op syntax.Token, yV starlark.Value, side starlark.Side) (starlark.Value, error) {
	x := gotime.Duration(d)
	var y gotime.Duration
	switch yV.(type) {
	case starlark.Int:
		i, ok := yV.(starlark.Int).Int64()
		if !ok {
			return nil, fmt.Errorf("duration binary operation: couldn't parse int")
		}
		y = gotime.Duration(i)
	case Duration:
		y = gotime.Duration(yV.(Duration))
	case Time:
		y := gotime.Time(yV.(Time))
		switch op {
		case syntax.PLUS:
			// duration + time = time
			return Time(y.Add(x)), nil
		case syntax.MINUS:
			// duration - time = duration
			return nil, nil
		}
	default:
		return nil, nil
	}

	switch op {
	case syntax.PLUS:
		return Duration(x + y), nil
	case syntax.MINUS:
		return Duration(x - y), nil
	case syntax.SLASH:
		if int64(y) == 0 {
			return nil, fmt.Errorf("cannot divide duration by zero")
		}
		return Duration(x / y), nil
	case syntax.STAR:
		return Duration(x * y), nil
	}

	return nil, nil
}

var durationMethods = map[string]builtinMethod{
	"hours":       durationhours,
	"minutes":     durationminutes,
	"seconds":     durationseconds,
	"nanoseconds": durationnanoseconds,
}

func durationhours(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	recv := gotime.Duration(recV.(Duration))
	return starlark.Float(recv.Hours()), nil
}

func durationminutes(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	recv := gotime.Duration(recV.(Duration))
	return starlark.Float(recv.Minutes()), nil
}

func durationseconds(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	recv := gotime.Duration(recV.(Duration))
	return starlark.Float(recv.Seconds()), nil
}

func durationnanoseconds(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	recv := gotime.Duration(recV.(Duration))
	return starlark.MakeInt64(recv.Nanoseconds()), nil
}

// Time is a starlark representation of a point in time
type Time gotime.Time

// String implements the Stringer interface
func (t Time) String() string { return gotime.Time(t).String() }

// Type returns a short string describing the value's type.
func (t Time) Type() string { return "time" }

// Freeze renders time immutable. required by starlark.Value interface
// because Time is already immutable this is a no-op
func (t Time) Freeze() {}

// Hash returns a function of x such that Equals(x, y) => Hash(x) == Hash(y)
// required by starlark.Value interface
func (t Time) Hash() (uint32, error) { return hashString(t.String()), nil }

// Truth returns the truth value of an object required by starlark.Value interface
func (t Time) Truth() starlark.Bool { return starlark.Bool(gotime.Time(t).IsZero()) }

// Attr gets a value for a string attribute, implementing dot expression support in starklark. required by starlark.HasAttrs interface
func (t Time) Attr(name string) (starlark.Value, error) { return builtinAttr(t, name, timeMethods) }

// AttrNames lists available dot expression strings for time. required by starlark.HasAttrs interface
func (t Time) AttrNames() []string { return builtinAttrNames(timeMethods) }

// CompareSameType implements comparison of two Time values. required by starlark.Comparable interface
func (t Time) CompareSameType(op syntax.Token, yV starlark.Value, depth int) (bool, error) {
	x := gotime.Time(t)
	y := gotime.Time(yV.(Time))
	cp := 0
	if x.Before(y) {
		cp = -1
	} else if x.After(y) {
		cp = 1
	}
	return threeway(op, cp), nil
}

// Binary implements binary operators, which satisfies the starlark.HasBinary interface
func (t Time) Binary(op syntax.Token, yV starlark.Value, side starlark.Side) (starlark.Value, error) {
	x := gotime.Time(t)

	switch yV.(type) {
	case Duration:
		y := gotime.Duration(yV.(Duration))
		switch op {
		// time + duration = time
		case syntax.PLUS:
			return Time(x.Add(y)), nil
		// time - duration = time
		case syntax.MINUS:
			return Time(x.Add(-y)), nil
		}
	case Time:
		y := gotime.Time(yV.(Time))
		switch op {
		// time - time = duration
		case syntax.MINUS:
			if side == starlark.Left {
				return Duration(x.Sub(y)), nil
			}
			return Duration(y.Sub(x)), nil
		}
	}

	// dunno, bail
	return nil, nil
}

var timeMethods = map[string]builtinMethod{
	"year":       timeyear,
	"month":      timemonth,
	"day":        timeday,
	"hour":       timehour,
	"minute":     timeminute,
	"second":     timesecond,
	"nanosecond": timenanosecond,
	"unix":       timeunix,
	"unix_nano":  timeunixnano,

	"in_location": timein,
	"format":      timeformat,
}

// TODO - consider using a higher order function to generate these
func timeyear(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	recv := gotime.Time(recV.(Time))
	return starlark.MakeInt(recv.Year()), nil
}

func timemonth(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	recv := gotime.Time(recV.(Time))
	return starlark.MakeInt(int(recv.Month())), nil
}

func timeday(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	recv := gotime.Time(recV.(Time))
	return starlark.MakeInt(recv.Day()), nil
}

func timehour(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	recv := gotime.Time(recV.(Time))
	return starlark.MakeInt(recv.Hour()), nil
}

func timeminute(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	recv := gotime.Time(recV.(Time))
	return starlark.MakeInt(recv.Minute()), nil
}

func timesecond(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	recv := gotime.Time(recV.(Time))
	return starlark.MakeInt(recv.Second()), nil
}

func timeunix(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	recv := gotime.Time(recV.(Time))
	return starlark.MakeInt64(recv.Unix()), nil
}

func timeunixnano(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	recv := gotime.Time(recV.(Time))
	return starlark.MakeInt64(recv.UnixNano()), nil
}

func timenanosecond(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	recv := gotime.Time(recV.(Time))
	return starlark.MakeInt(recv.Nanosecond()), nil
}

func timeformat(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var x starlark.String
	if err := starlark.UnpackArgs("format", args, kwargs, "x", &x); err != nil {
		return nil, err
	}

	recv := gotime.Time(recV.(Time))
	return starlark.String(recv.Format(string(x))), nil
}

func timein(fnname string, recV starlark.Value, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var x starlark.String
	if err := starlark.UnpackArgs("in_location", args, kwargs, "x", &x); err != nil {
		return nil, err
	}
	loc, err := gotime.LoadLocation(string(x))
	if err != nil {
		return nil, err
	}

	recv := gotime.Time(recV.(Time))
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

// hashString computes the FNV hash of s.
func hashString(s string) uint32 {
	var h uint32
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
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
