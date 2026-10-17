#!/bin/bash
#
# This script runs a small Starlark program in the cmd/starlark
# interpreter, with CPU profiling enabled, and prints the profile.
# Typical output, which reports Starlark source frames, is shown
# below

set -euo pipefail

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

cat <<'EOF' >"$tmpdir/demo.star"
def leaf(n):
    s = 0
    for i in range(n):
        s = (s + i) & 0xffff
    return s

def middle(n):
    return leaf(n) + leaf(n)

def top():
    for _ in range(50):
        middle(100000)

top()
EOF

go run ./cmd/starlark -cpuprofile="$tmpdir/cpu.prof" "$tmpdir/demo.star"

echo "=== pprof -top -lines ==="
go tool pprof -top -lines "$tmpdir/cpu.prof" | head -n 20

echo
echo "=== pprof -list=leaf ==="
go tool pprof -list=leaf "$tmpdir/cpu.prof"



# Output:
#
# === pprof -top -lines ===
# File: starlark
# Type: cpu
# Time: 2026-09-12 10:37:25 EDT
# Duration: 908.77ms, Total samples = 640ms (70.43%)
# Showing nodes accounting for 640ms, 100% of 640ms total
#       flat  flat%   sum%        cum   cum%
#      190ms 29.69% 29.69%      430ms 67.19%  leaf demo.star:4
#      110ms 17.19% 46.88%      160ms 25.00%  leaf demo.star:3
#       40ms  6.25% 53.12%       40ms  6.25%  go.starlark.net/starlark.(*rangeIterator).Next starlark/library.go:958
#       40ms  6.25% 59.38%       40ms  6.25%  runtime.pthread_cond_signal runtime/sys_darwin.go:491
#       30ms  4.69% 64.06%       40ms  6.25%  runtime.(*_panic).nextDefer runtime/panic.go:989
#       30ms  4.69% 68.75%       30ms  4.69%  runtime.(*_panic).start runtime/panic.go:909
#       20ms  3.12% 71.88%       20ms  3.12%  go.starlark.net/starlark.Int.get starlark/int_posix64.go:47
#       20ms  3.12% 75.00%       20ms  3.12%  runtime.(*_panic).nextDefer runtime/panic.go:945
#       10ms  1.56% 76.56%       10ms  1.56%  go.starlark.net/starlark.(*rangeIterator).Next starlark/library.go:956
#       10ms  1.56% 78.12%       10ms  1.56%  go.starlark.net/starlark.Binary starlark/eval.go:1071
#       10ms  1.56% 79.69%       70ms 10.94%  go.starlark.net/starlark.Binary starlark/eval.go:1073
#       10ms  1.56% 81.25%      120ms 18.75%  go.starlark.net/starlark.Binary starlark/eval.go:1139
#       10ms  1.56% 82.81%       40ms  6.25%  go.starlark.net/starlark.Binary starlark/eval.go:784
#       10ms  1.56% 84.38%       10ms  1.56%  go.starlark.net/starlark.Int.And starlark/int.go:279
#
# === pprof -list=leaf ===
# Total: 640ms
# ROUTINE ======================== leaf in demo.star
#      300ms      590ms (flat, cum) 92.19% of Total
#          .          .      1:def leaf(n):
#          .          .      2:    s = 0
#      110ms      160ms      3:    for i in range(n):
#      190ms      430ms      4:        s = (s + i) & 0xffff
#          .          .      5:    return s
#          .          .      6:
#          .          .      7:def middle(n):
#          .          .      8:    return leaf(n) + leaf(n)
#          .          .      9:
