// Copyright 2020 Honda Research Institute Europe GmbH. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package starlarktime defines functions to work with dates/times
package starlarktime

import (
	"fmt"
	"time"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Module time is a Starlark module similar to the datetime python package.
//
//   time = module(
//      parse,
//      now,
//      time,
//   )
//
// def parse(string, format[, location]):
//   The parse function parses the given "string" with the "format" specified.
//   Formats follow the specifications in the Golang "time" package.
//   The optional "location" argument can be used to specify the timezone
//   location for the resulting time. In case no location is given and the
//   given string does not contain any timezone information, the time is
//   returned in UTC.
//
//   For example
//     parse("2020-01-01T12:00:00Z", "2006-01-02T15:04:05Z", "Europe/Berlin")
//   will return a time object with the given time localized in the CET/CEST
//   timezone.
//
// def now([location]):
//   The now function returns the current time.
//   The optional "location" parameter can be used to specify the timezone
//   location for the returned time. In case no location is given, the local
//   time is returned.
//
// def time([year[, month[, day[, hour[, minute[, second[, nanosecond[, location]]]]]]]]):
//   The time function returns the a new time instance with the given values.
//   All parameters are optional and default to zero if not specified. The "location"
//   parameter can be used to specify the timezone location for the returned
//   time. It defaults to UTC if not specified.
//
var Module = &starlarkstruct.Module{
	Name: "time",
	Members: starlark.StringDict{
		"parse_time": starlark.NewBuiltin("time.parse", parseTime),
		"now":   starlark.NewBuiltin("time.now", now),
		"time":  starlark.NewBuiltin("time.time", newTime),
	},
}

func parseTime(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var value, fmt, tz starlark.String

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &value, &fmt, &tz); err != nil {
		return nil, err
	}

	var t time.Time
	var err error
    if len(fmt) < 1 {
        // Use RFC3339 by default
		t, err = time.Parse(time.RFC3339, string(value))
		if err != nil {
			return starlark.None, err
		}
    } else if len(tz) < 1 {
        // Use UTC by default
		t, err = time.Parse(string(fmt), string(value))
		if err != nil {
			return starlark.None, err
		}
	} else {
		loc, err := time.LoadLocation(string(tz))
		if err != nil {
			return starlark.None, err
		}
		t, err = time.ParseInLocation(string(fmt), string(value), loc)
		if err != nil {
			return starlark.None, err
		}
	}

	return &StarlarkTime{Time: t}, nil
}

func now(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var tz starlark.String

	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 0, &tz); err != nil {
		return nil, err
	}

	t := time.Now()
	if len(string(tz)) > 0 {
		loc, err := time.LoadLocation(string(tz))
		if err != nil {
			return starlark.None, err
		}
		t = t.In(loc)
	}

	return &StarlarkTime{Time: t}, nil
}

func newTime(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	y := starlark.MakeInt(0)
	m := starlark.MakeInt(0)
	d := starlark.MakeInt(0)
	h := starlark.MakeInt(0)
	min := starlark.MakeInt(0)
	s := starlark.MakeInt(0)
	ns := starlark.MakeInt(0)
	tz := starlark.String("UTC")
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 0, &y, &m, &d, &h, &min, &s, &ns, &tz); err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation(string(tz))
	if err != nil {
		return starlark.None, err
	}

	year, err := starlark.AsInt32(y)
	if err != nil {
		return starlark.None, fmt.Errorf("year: %v", err)
	}
	month, err := starlark.AsInt32(m)
	if err != nil {
		return starlark.None, fmt.Errorf("month: %v", err)
	}
	day, err := starlark.AsInt32(d)
	if err != nil {
		return starlark.None, fmt.Errorf("day: %v", err)
	}
	hour, err := starlark.AsInt32(h)
	if err != nil {
		return starlark.None, fmt.Errorf("hour: %v", err)
	}
	minute, err := starlark.AsInt32(min)
	if err != nil {
		return starlark.None, fmt.Errorf("minute: %v", err)
	}
	second, err := starlark.AsInt32(s)
	if err != nil {
		return starlark.None, fmt.Errorf("second: %v", err)
	}
	nanosecond, err := starlark.AsInt32(ns)
	if err != nil {
		return starlark.None, fmt.Errorf("nanosecond: %v", err)
	}

	t := time.Date(year, time.Month(month), day, hour, minute, second, nanosecond, loc)

	return &StarlarkTime{Time: t}, nil
}
