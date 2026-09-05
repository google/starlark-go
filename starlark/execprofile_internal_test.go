package starlark

// This file checks profiler details that the exported API does not show. For
// example, it checks that rows with equal sort values always use the same
// order.

import (
	"testing"

	"go.starlark.net/syntax"
)

func TestProfileRecordsTotalOrder(t *testing.T) {
	firstFilename, secondFilename := "dup.star", "dup.star"
	positions := []syntax.Position{
		callablePosition(NewBuiltin("dup", nil)),
		syntax.MakePosition(&firstFilename, 1, 1),
		syntax.MakePosition(&secondFilename, 1, 1),
	}

	var p ExecProfile
	p.records = make(map[profileKey]*profileRecord)
	for i, address := range []uintptr{0x30, 0x10, 0x20} {
		key := profileKey{address: address, name: "dup"}
		p.records[key] = &profileRecord{
			owner:          &p,
			key:            key,
			position:       positions[i],
			completedCalls: int64(i + 1),
		}
	}

	first := p.Records()
	for i := range 50 {
		got := p.Records()
		for j := range got {
			if got[j] != first[j] {
				t.Fatalf("render %d record %d = %+v, first render had %+v", i, j, got[j], first[j])
			}
		}
	}
}
