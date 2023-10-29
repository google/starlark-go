package time

import (
	"errors"
	"testing"
	"time"

	"go.starlark.net/starlark"
)

func TestPerThreadNowReturnsCorrectTime(t *testing.T) {
	th := &starlark.Thread{}
	date := time.Date(1, 2, 3, 4, 5, 6, 7, time.UTC)
	SetNow(th, func() (time.Time, error) {
		return date, nil
	})

	res, err := starlark.Call(th, Module.Members["now"], nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	retTime := time.Time(res.(Time))

	if !retTime.Equal(date) {
		t.Fatal("Expected time to be equal", retTime, date)
	}
}

func TestPerThreadNowReturnsError(t *testing.T) {
	th := &starlark.Thread{}
	e := errors.New("no time")
	SetNow(th, func() (time.Time, error) {
		return time.Time{}, e
	})

	_, err := starlark.Call(th, Module.Members["now"], nil, nil)
	if !errors.Is(err, e) {
		t.Fatal("Expected equal error", e, err)
	}
}

func TestGlobalNowReturnsCorrectTime(t *testing.T) {
	th := &starlark.Thread{}

	oldNow := NowFunc
	defer func() {
		NowFunc = oldNow
	}()

	date := time.Date(1, 2, 3, 4, 5, 6, 7, time.UTC)
	NowFunc = func() time.Time {
		return date
	}

	res, err := starlark.Call(th, Module.Members["now"], nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	retTime := time.Time(res.(Time))

	if !retTime.Equal(date) {
		t.Fatal("Expected time to be equal", retTime, date)
	}
}

func TestGlobalNowReturnsErrorWhenNil(t *testing.T) {
	th := &starlark.Thread{}

	oldNow := NowFunc
	defer func() {
		NowFunc = oldNow
	}()

	NowFunc = nil

	_, err := starlark.Call(th, Module.Members["now"], nil, nil)
	if err == nil {
		t.Fatal("Expected to get an error")
	}
}
