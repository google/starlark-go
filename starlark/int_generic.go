//go:build (!linux && !darwin && !dragonfly && !freebsd && !netbsd && !solaris) || (!amd64 && !arm64 && !mips64x && !ppc64x && !loong64) || noposixint
// +build !linux,!darwin,!dragonfly,!freebsd,!netbsd,!solaris !amd64,!arm64,!mips64x,!ppc64x,!loong64 noposixint

package starlark

import "math/big"

const hasPosixInts = false

// int_get returns the (small, big) arms of the union.
func int_get(i Int) (int64, *big.Int) {
	switch i := i.(type) {
	case intSmall:
		return int64(i), nil
	case *intBig:
		return 0, (*big.Int)(i)
	default:
		panic("Int is not an int?")
	}
}

// Precondition: x cannot be represented as int32.
func makeBigInt(x *big.Int) Int { return (*intBig)(x) }

// Precondition: math.MinInt32 <= x && x <= math.MaxInt32
func makeSmallInt(x int64) Int { return intSmall(x) }
