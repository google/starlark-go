//go:build (linux || darwin || dragonfly || freebsd || netbsd || solaris) && (amd64 || arm64 || mips64x || ppc64x || loong64) && !noposixint
// +build linux darwin dragonfly freebsd netbsd solaris
// +build amd64 arm64 mips64x ppc64x loong64
// +build !noposixint

package starlark

// This file defines an optimized Int implementation for 64-bit machines
// running POSIX. It reserves a 4GB portion of the address space using
// mmap and represents int32 values as addresses within that range. This
// disambiguates int32 values from *big.Int pointers, letting all Int
// values be represented as an unsafe.Pointer, so that Int-to-Value
// interface conversion need not allocate.

// Although iOS (which, like macOS, appears as darwin/arm64) is
// POSIX-compliant, it limits each process to about 700MB of virtual
// address space, which defeats the optimization.  Similarly,
// OpenBSD's default ulimit for virtual memory is a measly GB or so.
// On both those platforms the attempted optimization will fail and
// fall back to the slow implementation.

// An alternative approach to this optimization would be to embed the
// int32 values in pointers using odd values, which can be distinguished
// from (even) *big.Int pointers. However, the Go runtime does not allow
// user programs to manufacture pointers to arbitrary locations such as
// within the zero page, or non-span, non-mmap, non-stack locations,
// and it may panic if it encounters them; see Issue #382.

import (
	"fmt"
	"log"
	"math"
	"math/big"
	"strconv"
	"unsafe"

	"go.starlark.net/syntax"
	"golang.org/x/sys/unix"
)

// Int represents a union of (int32, *big.Int) in a single pointer,
// so that Int-to-Value conversions need not allocate.
//
// The pointer is either a *big.Int, if the value is big, or a pointer into a
// reserved portion of the address space (smallints), if the value is small
// and the address space allocation succeeded.
//
// See int_generic.go for the basic representation concepts.
type intPosix64 struct {
	impl unsafe.Pointer
}

const hasPosixInts = true

func (i intPosix64) get() (int64, *big.Int) {
	if ptr := uintptr(i.impl); ptr >= smallints && ptr < smallints+1<<32 {
		return math.MinInt32 + int64(ptr-smallints), nil
	}
	return 0, (*big.Int)(i.impl)
}

func (i intPosix64) Unary(op syntax.Token) (Value, error) {
	switch op {
	case syntax.MINUS:
		return zero.Sub(i), nil
	case syntax.PLUS:
		return i, nil
	case syntax.TILDE:
		return i.Not(), nil
	}
	return nil, nil
}

func (i intPosix64) Int64() (_ int64, ok bool) {
	iSmall, iBig := i.get()
	if iBig != nil {
		x, acc := bigintToInt64(iBig)
		if acc != big.Exact {
			return // inexact
		}
		return x, true
	}
	return iSmall, true
}

func (i intPosix64) BigInt() *big.Int {
	iSmall, iBig := i.get()
	if iBig != nil {
		return new(big.Int).Set(iBig)
	}
	return big.NewInt(iSmall)
}

func (i intPosix64) bigInt() *big.Int {
	iSmall, iBig := i.get()
	if iBig != nil {
		return iBig
	}
	return big.NewInt(iSmall)
}

func (i intPosix64) Uint64() (_ uint64, ok bool) {
	iSmall, iBig := i.get()
	if iBig != nil {
		x, acc := bigintToUint64(iBig)
		if acc != big.Exact {
			return // inexact
		}
		return x, true
	}
	if iSmall < 0 {
		return // inexact
	}
	return uint64(iSmall), true
}

// smallints is the base address of a 2^32 byte memory region.
// Pointers to addresses in this region represent int32 values.
// We assume smallints is not at the very top of the address space.
//
// Zero means the optimization is disabled and all Ints allocate a big.Int.
var smallints = reserveAddresses(1 << 32)

func reserveAddresses(len int) uintptr {
	b, err := unix.Mmap(-1, 0, len, unix.PROT_READ, unix.MAP_PRIVATE|unix.MAP_ANON)
	if err != nil {
		log.Printf("Starlark failed to allocate 4GB address space: %v. Integer performance may suffer.", err)
		return 0 // optimization disabled
	}
	return uintptr(unsafe.Pointer(&b[0]))
}

func (i intPosix64) Format(s fmt.State, ch rune) {
	iSmall, iBig := i.get()
	if iBig != nil {
		iBig.Format(s, ch)
		return
	}
	big.NewInt(iSmall).Format(s, ch)
}
func (i intPosix64) String() string {
	iSmall, iBig := i.get()
	if iBig != nil {
		return iBig.Text(10)
	}
	return strconv.FormatInt(iSmall, 10)
}
func (i intPosix64) Type() string { return "int" }
func (i intPosix64) Freeze()      {} // immutable
func (i intPosix64) Truth() Bool  { return i.Sign() != 0 }
func (i intPosix64) Hash() (uint32, error) {
	iSmall, iBig := i.get()
	var lo big.Word
	if iBig != nil {
		lo = iBig.Bits()[0]
	} else {
		lo = big.Word(iSmall)
	}
	return 12582917 * uint32(lo+3), nil
}
func (x intPosix64) CompareSameType(op syntax.Token, v Value, depth int) (bool, error) {
	y := v.(Int)
	xSmall, xBig := x.get()
	ySmall, yBig := int_get(y)
	if xBig != nil || yBig != nil {
		return threeway(op, x.bigInt().Cmp(y.bigInt())), nil
	}
	return threeway(op, signum64(xSmall-ySmall)), nil
}

// Float returns the float value nearest i.
func (i intPosix64) Float() Float {
	iSmall, iBig := i.get()
	if iBig != nil {
		// Fast path for hardware int-to-float conversions.
		if iBig.IsUint64() {
			return Float(iBig.Uint64())
		} else if iBig.IsInt64() {
			return Float(iBig.Int64())
		}

		f, _ := new(big.Float).SetInt(iBig).Float64()
		return Float(f)
	}
	return Float(iSmall)
}

// finiteFloat returns the finite float value nearest i,
// or an error if the magnitude is too large.
func (i intPosix64) finiteFloat() (Float, error) {
	f := i.Float()
	if math.IsInf(float64(f), 0) {
		return 0, fmt.Errorf("int too large to convert to float")
	}
	return f, nil
}

func (x intPosix64) Sign() int {
	xSmall, xBig := x.get()
	if xBig != nil {
		return xBig.Sign()
	}
	return signum64(xSmall)
}

func (x intPosix64) Add(y Int) Int {
	xSmall, xBig := x.get()
	ySmall, yBig := int_get(y)
	if xBig != nil || yBig != nil {
		return MakeBigInt(new(big.Int).Add(x.bigInt(), y.bigInt()))
	}
	return MakeInt64(xSmall + ySmall)
}
func (x intPosix64) Sub(y Int) Int {
	xSmall, xBig := x.get()
	ySmall, yBig := int_get(y)
	if xBig != nil || yBig != nil {
		return MakeBigInt(new(big.Int).Sub(x.bigInt(), y.bigInt()))
	}
	return MakeInt64(xSmall - ySmall)
}
func (x intPosix64) Mul(y Int) Int {
	xSmall, xBig := x.get()
	ySmall, yBig := int_get(y)
	if xBig != nil || yBig != nil {
		return MakeBigInt(new(big.Int).Mul(x.bigInt(), y.bigInt()))
	}
	return MakeInt64(xSmall * ySmall)
}
func (x intPosix64) Or(y Int) Int {
	xSmall, xBig := x.get()
	ySmall, yBig := int_get(y)
	if xBig != nil || yBig != nil {
		return MakeBigInt(new(big.Int).Or(x.bigInt(), y.bigInt()))
	}
	return makeSmallInt(xSmall | ySmall)
}
func (x intPosix64) And(y Int) Int {
	xSmall, xBig := x.get()
	ySmall, yBig := int_get(y)
	if xBig != nil || yBig != nil {
		return MakeBigInt(new(big.Int).And(x.bigInt(), y.bigInt()))
	}
	return makeSmallInt(xSmall & ySmall)
}
func (x intPosix64) Xor(y Int) Int {
	xSmall, xBig := x.get()
	ySmall, yBig := int_get(y)
	if xBig != nil || yBig != nil {
		return MakeBigInt(new(big.Int).Xor(x.bigInt(), y.bigInt()))
	}
	return makeSmallInt(xSmall ^ ySmall)
}
func (x intPosix64) Not() Int {
	xSmall, xBig := x.get()
	if xBig != nil {
		return MakeBigInt(new(big.Int).Not(xBig))
	}
	return makeSmallInt(^xSmall)
}
func (x intPosix64) Lsh(y uint) Int { return MakeBigInt(new(big.Int).Lsh(x.bigInt(), y)) }
func (x intPosix64) Rsh(y uint) Int { return MakeBigInt(new(big.Int).Rsh(x.bigInt(), y)) }

// Precondition: y is nonzero.
func (x intPosix64) Div(y Int) Int {
	xSmall, xBig := x.get()
	ySmall, yBig := int_get(y)
	// http://python-history.blogspot.com/2010/08/why-pythons-integer-division-floors.html
	if xBig != nil || yBig != nil {
		return int_div_big(x.bigInt(), y.bigInt())
	}

	return int_div_small(xSmall, ySmall)
}

// Precondition: y is nonzero.
func (x intPosix64) Mod(y Int) Int {
	xSmall, xBig := x.get()
	ySmall, yBig := int_get(y)
	if xBig != nil || yBig != nil {
		return int_mod_big(x.bigInt(), y.bigInt())
	}

	return int_mod_small(xSmall, ySmall)
}

func (i intPosix64) rational() *big.Rat {
	iSmall, iBig := i.get()
	if iBig != nil {
		return new(big.Rat).SetInt(iBig)
	}
	return new(big.Rat).SetInt64(iSmall)
}

// int_get returns the (small, big) arms of the union.
func int_get(i Int) (int64, *big.Int) {
	switch i := i.(type) {
	case intSmall:
		return int64(i), nil
	case *intBig:
		return 0, (*big.Int)(i)
	case intPosix64:
		return i.get()
	default:
		panic("Int is not an int?")
	}
}

// Precondition: x cannot be represented as int32.
func makeBigInt(x *big.Int) Int {
	if smallints == 0 {
		return (*intBig)(x)
	}
	return intPosix64{unsafe.Pointer(x)}
}

// Precondition: math.MinInt32 <= x && x <= math.MaxInt32
func makeSmallInt(x int64) Int {
	if smallints == 0 {
		// optimization disabled
		return intSmall(x)
	}

	return intPosix64{unsafe.Pointer(uintptr(x-math.MinInt32) + smallints)}
}
