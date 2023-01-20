// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package starlark

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strconv"

	"go.starlark.net/syntax"
)

// Int is the type of a Starlark int.
type Int interface {
	Value

	// Private part
	finiteFloat() (Float, error)
	rational() *big.Rat

	CompareSameType(op syntax.Token, y Value, depth int) (bool, error)

	// Int64 returns the value as an int64.
	// If it is not exactly representable the result is undefined and ok is false.
	Int64() (int64, bool)

	// BigInt returns a new big.Int with the same value as the Int.
	BigInt() *big.Int

	// bigInt returns the value as a big.Int.
	// It differs from BigInt in that this method returns the actual
	// reference and any modification will change the state of i.
	bigInt() *big.Int

	// Uint64 returns the value as a uint64.
	// If it is not exactly representable the result is undefined and ok is false.
	Uint64() (uint64, bool)

	Float() Float
	Format(s fmt.State, ch rune)
	Not() Int
	Sign() int
	Add(y Int) Int
	Sub(y Int) Int
	Mul(y Int) Int
	Or(y Int) Int
	Xor(y Int) Int
	And(y Int) Int
	Lsh(y uint) Int
	Rsh(y uint) Int
	// Precondition: y is nonzero.
	Div(y Int) Int

	// Precondition: y is nonzero.
	Mod(y Int) Int

	// Unary implements the operations +int, -int, and ~int.
	Unary(op syntax.Token) (Value, error)
}

var (
	_ HasUnary   = Int(nil)
	_ Comparable = Int(nil)
)

// --- high-level accessors ---

// MakeInt returns a Starlark int for the specified signed integer.
func MakeInt(x int) Int { return MakeInt64(int64(x)) }

// MakeInt64 returns a Starlark int for the specified int64.
func MakeInt64(x int64) Int {
	if math.MinInt32 <= x && x <= math.MaxInt32 {
		return makeSmallInt(x)
	}
	return makeBigInt(big.NewInt(x))
}

// MakeUint returns a Starlark int for the specified unsigned integer.
func MakeUint(x uint) Int { return MakeUint64(uint64(x)) }

// MakeUint64 returns a Starlark int for the specified uint64.
func MakeUint64(x uint64) Int {
	if x <= math.MaxInt32 {
		return makeSmallInt(int64(x))
	}
	return makeBigInt(new(big.Int).SetUint64(x))
}

// MakeBigInt returns a Starlark int for the specified big.Int.
// The new Int value will contain a copy of x. The caller is safe to modify x.
func MakeBigInt(x *big.Int) Int {
	if isSmall(x) {
		return makeSmallInt(x.Int64())
	}
	z := new(big.Int).Set(x)
	return makeBigInt(z)
}

func isSmall(x *big.Int) bool {
	n := x.BitLen()
	return n < 32 || n == 32 && x.Int64() == math.MinInt32
}

var (
	zero, one = makeSmallInt(0), makeSmallInt(1)
	oneBig    = big.NewInt(1)
)

// The math/big API should provide this function.
func bigintToInt64(i *big.Int) (int64, big.Accuracy) {
	sign := i.Sign()
	if sign > 0 {
		if i.Cmp(maxint64) > 0 {
			return math.MaxInt64, big.Below
		}
	} else if sign < 0 {
		if i.Cmp(minint64) < 0 {
			return math.MinInt64, big.Above
		}
	}
	return i.Int64(), big.Exact
}

// The math/big API should provide this function.
func bigintToUint64(i *big.Int) (uint64, big.Accuracy) {
	sign := i.Sign()
	if sign > 0 {
		if i.BitLen() > 64 {
			return math.MaxUint64, big.Below
		}
	} else if sign < 0 {
		return 0, big.Above
	}
	return i.Uint64(), big.Exact
}

var (
	minint64 = new(big.Int).SetInt64(math.MinInt64)
	maxint64 = new(big.Int).SetInt64(math.MaxInt64)
)

// AsInt32 returns the value of x if is representable as an int32.
func AsInt32(x Value) (int, error) {
	i, ok := x.(Int)
	if !ok {
		return 0, fmt.Errorf("got %s, want int", x.Type())
	}
	iSmall, iBig := int_get(i)
	if iBig != nil {
		return 0, fmt.Errorf("%s out of range", i)
	}
	return int(iSmall), nil
}

// AsInt sets *ptr to the value of Starlark int x, if it is exactly representable,
// otherwise it returns an error.
// The type of ptr must be one of the pointer types *int, *int8, *int16, *int32, or *int64,
// or one of their unsigned counterparts including *uintptr.
func AsInt(x Value, ptr interface{}) error {
	xint, ok := x.(Int)
	if !ok {
		return fmt.Errorf("got %s, want int", x.Type())
	}

	bits := reflect.TypeOf(ptr).Elem().Size() * 8
	switch ptr.(type) {
	case *int, *int8, *int16, *int32, *int64:
		i, ok := xint.Int64()
		if !ok || bits < 64 && !(-1<<(bits-1) <= i && i < 1<<(bits-1)) {
			return fmt.Errorf("%s out of range (want value in signed %d-bit range)", xint, bits)
		}
		switch ptr := ptr.(type) {
		case *int:
			*ptr = int(i)
		case *int8:
			*ptr = int8(i)
		case *int16:
			*ptr = int16(i)
		case *int32:
			*ptr = int32(i)
		case *int64:
			*ptr = int64(i)
		}

	case *uint, *uint8, *uint16, *uint32, *uint64, *uintptr:
		i, ok := xint.Uint64()
		if !ok || bits < 64 && i >= 1<<bits {
			return fmt.Errorf("%s out of range (want value in unsigned %d-bit range)", xint, bits)
		}
		switch ptr := ptr.(type) {
		case *uint:
			*ptr = uint(i)
		case *uint8:
			*ptr = uint8(i)
		case *uint16:
			*ptr = uint16(i)
		case *uint32:
			*ptr = uint32(i)
		case *uint64:
			*ptr = uint64(i)
		case *uintptr:
			*ptr = uintptr(i)
		}
	default:
		panic(fmt.Sprintf("invalid argument type: %T", ptr))
	}
	return nil
}

// NumberToInt converts a number x to an integer value.
// An int is returned unchanged, a float is truncated towards zero.
// NumberToInt reports an error for all other values.
func NumberToInt(x Value) (Int, error) {
	switch x := x.(type) {
	case Int:
		return x, nil
	case Float:
		f := float64(x)
		if math.IsInf(f, 0) {
			return zero, fmt.Errorf("cannot convert float infinity to integer")
		} else if math.IsNaN(f) {
			return zero, fmt.Errorf("cannot convert float NaN to integer")
		}
		return finiteFloatToInt(x), nil

	}
	return zero, fmt.Errorf("cannot convert %s to int", x.Type())
}

// finiteFloatToInt converts f to an Int, truncating towards zero.
// f must be finite.
func finiteFloatToInt(f Float) Int {
	// We avoid '<= MaxInt64' so that both constants are exactly representable as floats.
	// See https://github.com/google/starlark-go/issues/375.
	if math.MinInt64 <= f && f < math.MaxInt64+1 {
		// small values
		return MakeInt64(int64(f))
	}
	rat := f.rational()
	if rat == nil {
		panic(f) // non-finite
	}
	return MakeBigInt(new(big.Int).Div(rat.Num(), rat.Denom()))
}

func int_hash(lo big.Word) (uint32, error) { return 12582917 * uint32(lo+3), nil }

func int_compare_big(x, y *big.Int, op syntax.Token) (bool, error) {
	return threeway(op, x.Cmp(y)), nil
}

func int_compare_small(x, y int64, op syntax.Token) (bool, error) {
	return threeway(op, signum64(x-y)), nil
}

// Precondition: y is nonzero.
func int_div_small(x, y int64) Int {
	quo := x / y
	rem := x % y
	if (x < 0) != (y < 0) && rem != 0 {
		quo -= 1
	}
	return MakeInt64(quo)
}

func int_div_big(xb, yb *big.Int) Int {
	var quo, rem big.Int
	quo.QuoRem(xb, yb, &rem)
	if (xb.Sign() < 0) != (yb.Sign() < 0) && rem.Sign() != 0 {
		quo.Sub(&quo, oneBig)
	}
	return MakeBigInt(&quo)
}

// Precondition: y is nonzero.
func int_mod_small(x, y int64) Int {
	rem := x % y
	if (x < 0) != (y < 0) && rem != 0 {
		rem += y
	}
	return makeSmallInt(rem)
}

func int_mod_big(xb, yb *big.Int) Int {
	var quo, rem big.Int
	quo.QuoRem(xb, yb, &rem)
	if (xb.Sign() < 0) != (yb.Sign() < 0) && rem.Sign() != 0 {
		rem.Add(&rem, yb)
	}
	return MakeBigInt(&rem)
}

type intBig big.Int

var _ Int = &intBig{}

func (*intBig) Freeze()      {}
func (*intBig) Type() string { return "int" }

// Value interface
func (x *intBig) String() string        { return (*big.Int)(x).Text(10) }
func (x *intBig) Truth() Bool           { return true }
func (x *intBig) Hash() (uint32, error) { return int_hash((*big.Int)(x).Bits()[0]) }

// Unary
func (x *intBig) BigInt() *big.Int { return new(big.Int).Set((*big.Int)(x)) }

func (x *intBig) Unary(op syntax.Token) (Value, error) {
	switch op {
	case syntax.MINUS:
		return zero.Sub(x), nil
	case syntax.PLUS:
		return x, nil
	case syntax.TILDE:
		return x.Not(), nil
	}
	return nil, nil
}

func (x *intBig) Int64() (_ int64, ok bool) {
	i, acc := bigintToInt64((*big.Int)(x))
	if acc != big.Exact {
		return // inexact
	}
	return i, true
}

func (x *intBig) Uint64() (_ uint64, ok bool) {
	i, acc := bigintToUint64((*big.Int)(x))
	if acc != big.Exact {
		return // inexact
	}
	return i, true
}

func (x *intBig) Float() Float {
	f, _ := new(big.Float).SetInt((*big.Int)(x)).Float64()
	return Float(f)
}

func (x *intBig) Format(s fmt.State, ch rune) {
	(*big.Int)(x).Format(s, ch)
}

func (x *intBig) Not() Int {
	return makeBigInt(new(big.Int).Not((*big.Int)(x)))
}

func (x *intBig) bigInt() *big.Int {
	return (*big.Int)(x)
}

// Binary
func (x *intBig) Add(y Int) Int  { return MakeBigInt(new(big.Int).Add(x.bigInt(), y.bigInt())) }
func (x *intBig) And(y Int) Int  { return MakeBigInt(new(big.Int).And(x.bigInt(), y.bigInt())) }
func (x *intBig) Div(y Int) Int  { return int_div_big((*big.Int)(x), y.bigInt()) }
func (x *intBig) Mod(y Int) Int  { return int_mod_big((*big.Int)(x), y.bigInt()) }
func (x *intBig) Mul(y Int) Int  { return MakeBigInt(new(big.Int).Mul(x.bigInt(), y.bigInt())) }
func (x *intBig) Or(y Int) Int   { return MakeBigInt(new(big.Int).Or(x.bigInt(), y.bigInt())) }
func (x *intBig) Sign() int      { return (*big.Int)(x).Sign() }
func (x *intBig) Sub(y Int) Int  { return MakeBigInt(new(big.Int).Sub(x.bigInt(), y.bigInt())) }
func (x *intBig) Xor(y Int) Int  { return MakeBigInt(new(big.Int).Xor(x.bigInt(), y.bigInt())) }
func (x *intBig) Lsh(y uint) Int { return makeBigInt(new(big.Int).Lsh((*big.Int)(x), y)) }
func (x *intBig) Rsh(y uint) Int { return MakeBigInt(new(big.Int).Rsh((*big.Int)(x), y)) }

func (x *intBig) CompareSameType(op syntax.Token, y Value, depth int) (bool, error) {
	return int_compare_big(x.bigInt(), y.(Int).bigInt(), op)
}

func (x *intBig) finiteFloat() (Float, error) {
	f := x.Float()
	if math.IsInf(float64(f), 0) {
		return 0, fmt.Errorf("int too large to convert to float")
	}
	return f, nil
}

func (x *intBig) rational() *big.Rat {
	return new(big.Rat).SetInt((*big.Int)(x))
}

type intSmall int64

var _ Int = intSmall(0)

func (intSmall) Freeze()      {}
func (intSmall) Type() string { return "int" }

func (x intSmall) Hash() (uint32, error) {
	return int_hash(big.Word(x))
}

func (x intSmall) String() string {
	return strconv.FormatInt(int64(x), 10)
}

func (x intSmall) Truth() Bool {
	return x != 0
}

func (x intSmall) Unary(op syntax.Token) (Value, error) {
	switch op {
	case syntax.MINUS:
		return -x, nil
	case syntax.PLUS:
		return x, nil
	case syntax.TILDE:
		return ^x, nil
	}
	return nil, nil
}

func (x intSmall) CompareSameType(op syntax.Token, y Value, depth int) (bool, error) {
	ySmall, yBig := int_get(y.(Int))
	if yBig != nil {
		return int_compare_big(x.bigInt(), yBig, op)
	}

	return int_compare_small(int64(x), ySmall, op)
}

func (x intSmall) And(y Int) Int {
	ySmall, yBig := int_get(y)
	if yBig != nil {
		return MakeBigInt(new(big.Int).And(x.bigInt(), yBig))
	}
	return makeSmallInt(int64(x) & ySmall)
}

func (x intSmall) BigInt() *big.Int { return x.bigInt() }

func (x intSmall) Add(y Int) Int {
	ySmall, yBig := int_get(y)
	if yBig != nil {
		return MakeBigInt(new(big.Int).Add(x.bigInt(), yBig))
	}

	return MakeInt64(int64(x) + ySmall)
}

func (x intSmall) Div(y Int) Int {
	ySmall, yBig := int_get(y)
	if yBig != nil {
		return int_div_big(x.bigInt(), yBig)
	}

	return int_div_small(int64(x), ySmall)
}

func (x intSmall) Format(s fmt.State, ch rune) {
	big.NewInt(int64(x)).Format(s, ch)
}

func (x intSmall) Int64() (int64, bool) {
	return int64(x), true
}

func (x intSmall) Uint64() (_ uint64, ok bool) {
	if x < 0 {
		return // inexact
	} else {
		return uint64(x), true
	}
}

func (x intSmall) Float() Float {
	return Float(x)
}

func (x intSmall) Lsh(y uint) Int {
	return MakeBigInt(new(big.Int).Lsh(x.bigInt(), y))
}

func (x intSmall) Mod(y Int) Int {
	ySmall, yBig := int_get(y)
	if yBig != nil {
		return int_mod_big(x.bigInt(), yBig)
	}

	return int_mod_small(int64(x), ySmall)
}

func (x intSmall) Mul(y Int) Int {
	ySmall, yBig := int_get(y)
	if yBig != nil {
		return MakeBigInt(new(big.Int).Mul(x.bigInt(), yBig))
	}
	return MakeInt64(int64(x) * ySmall)
}

func (x intSmall) Not() Int { return ^x }

func (x intSmall) Or(y Int) Int {
	ySmall, yBig := int_get(y)
	if yBig != nil {
		return MakeBigInt(new(big.Int).Or(x.bigInt(), yBig))
	}
	return makeSmallInt(int64(x) | ySmall)
}

func (x intSmall) Xor(y Int) Int {
	ySmall, yBig := int_get(y)
	if yBig != nil {
		return MakeBigInt(new(big.Int).Xor(x.bigInt(), yBig))
	}

	return makeSmallInt(int64(x) ^ ySmall)
}

func (x intSmall) Rsh(y uint) Int {
	return x >> y
}

func (x intSmall) Sign() int {
	return signum64(int64(x))
}

func (x intSmall) Sub(y Int) Int {
	ySmall, yBig := int_get(y)
	if yBig != nil {
		return MakeBigInt(new(big.Int).Sub(x.bigInt(), y.bigInt()))
	}
	return MakeInt64(int64(x) - ySmall)
}

func (x intSmall) bigInt() *big.Int {
	return big.NewInt(int64(x))
}

func (x intSmall) finiteFloat() (Float, error) {
	// FIXME check if right
	return x.Float(), nil
}

func (x intSmall) rational() *big.Rat {
	return new(big.Rat).SetInt64(int64(x))
}
