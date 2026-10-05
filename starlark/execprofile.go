package starlark

// ExecProfile measures Starlark calls on one thread at a time. It does not
// profile all threads at once. Use StartProfile to profile all threads.
//
// ExecProfile measures time at four points in the interpreter:
//
//	recordCallPush  a call frame was added (Call)
//	recordCallPop   a call frame will be removed (Call cleanup)
//	recordPause     the top frame will wait for another call or a load
//	recordResume    the top frame is running again (CALL and LOAD)
//
// A span is a period when one frame is the innermost frame running Starlark
// code. A thread can have only one open span. Spans do not overlap, so their
// total time cannot be longer than the full execution time. A frame's spans
// add up to its Self time.
//
// Push and pop must also end spans. Go code in a built-in can call Starlark
// without a CALL opcode. Push stops the built-in's span, so the called
// function's time is not part of the built-in's Self. Pop starts the
// built-in's span again because no opcode will do it. The frame records
// whether a span was open before push. This keeps loads and normal calls
// paused when needed.
//
// Each point first checks whether profiling is active and returns immediately
// when it is not.

import (
	"cmp"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
	"time"
	"unsafe"

	"go.starlark.net/syntax"
)

// An ExecProfile records the exact wall time of Starlark calls on one thread.
// Assign one to [Thread.Profile] before execution:
//
//	var p starlark.ExecProfile
//	thread.Profile = &p
//	_, err := starlark.ExecFile(thread, filename, src, nil)
//	p.WriteText(os.Stderr)
//
// The zero value is ready to use. It keeps data from each execution until it
// is replaced or cleared. There is no reset method. Use a new ExecProfile to
// start with no data. Do not copy an ExecProfile value after its first use;
// the same pointer may be reused as described below.
//
// # Identity
//
// Each row represents one function. ExecProfile identifies the function by
// its address, name, and definition position. More than one row can have the
// same name. For example, every module's top-level code is named
// "<toplevel>". Use both Name and Position when looking for a row. A callable
// with no definition position, such as a built-in, uses "<builtin>". Built-ins
// with different names or implementations use separate rows.
//
// Two calls use the same row only when all three values match. This groups
// repeated calls to one function. It also groups calls to the same compiled
// [Program] when [ExecProfile.Merge] combines profiles from several threads.
// A later execution of a file can also use the same row if Go freed the old
// compiled code and reused its address. Other functions from separate
// compilations use separate rows. ExecProfile does not keep a Callable in
// memory. It copies only the address, name, and position.
//
// # Threads
//
// ExecProfile follows the same goroutine rules as [Thread.Print] and
// [Thread.Load]. Set or clear it only from the goroutine that runs the thread.
// Read it only after execution has returned to the calling Go code.
//
// You can attach the same *ExecProfile to several threads if their executions
// do not overlap. For example, a Load function can create a child thread while
// its parent thread waits. Do not attach the same *ExecProfile to threads that
// run at the same time. This can produce wrong profile data and may stop the
// process with "fatal error: concurrent map writes". For parallel executions,
// use one ExecProfile per thread. Combine them later with [ExecProfile.Merge].
//
// # Attaching during execution
//
// You can attach, remove, or replace an ExecProfile while code is running.
// A profile records a call only if it was attached when the call started.
// A profile attached later does not count a call that is already running.
// That call has no Calls, Cum, or Max. It keeps only the Self time recorded
// while the profile was attached.
//
// Changing the field cannot end the open span at that exact time. When the
// span ends, its time is added to the old profile.
//
// # Cost
//
// Profiling adds clock reads and a map lookup to each call. After a function's
// first call, later calls do not allocate.
//
// A thread with no ExecProfile does a few field checks for each call. It does
// not make extra function calls. No background work continues after the last
// reference to an ExecProfile is removed.
//
// See also [StartProfile]. It samples all threads and writes a profile in
// pprof format. The two profilers can run at the same time. Their results
// differ when a built-in calls Starlark. Only ExecProfile leaves the called
// Starlark function's time out of the built-in's Self.
type ExecProfile struct {
	records map[profileKey]*profileRecord
}

// A profile key needs the address, name, and definition position. The address
// alone is not enough. Two Builtins can use the same implementation but have
// different names. Also, callables with no available address all use zero.
//
// The address and name are still not enough. Go can free compiled code and
// reuse its funcode address. Without the position, functions with the same
// name from separate compilations could use one row. The key copies a filename
// and two numbers from the callable. It does not keep the callable in memory.
//
// Store the parts of the position instead of a syntax.Position. A
// syntax.Position stores its filename through a pointer, so equal filenames
// could compare as different pointers.
type profileKey struct {
	address  uintptr
	name     string
	filename string
	line     int32
	column   int32
}

type profileRecord struct {
	owner          *ExecProfile
	key            profileKey
	position       syntax.Position
	completedCalls int64
	selfTime       time.Duration
	cumulativeTime time.Duration
	maxTime        time.Duration
	activeCalls    int
}

type profileSpan struct {
	record    *profileRecord
	startedAt int64 // zero when closed
}

func (s *profileSpan) isOpen() bool { return s.startedAt != 0 }

func (s *profileSpan) close(now int64) {
	if !s.isOpen() {
		return
	}
	s.record.selfTime += time.Duration(now - s.startedAt)
	s.record = nil
	s.startedAt = 0
}

func (s *profileSpan) open(record *profileRecord, now int64) {
	s.record = record
	s.startedAt = now
}

// Do not store c. It could keep a lambda's free variables in memory as long as
// the profile remains in memory.
func (p *ExecProfile) recordForCallable(c Callable) *profileRecord {
	address, position := profileIdentity(c)
	key := profileKey{
		address:  address,
		name:     c.Name(),
		filename: position.Filename(),
		line:     position.Line,
		column:   position.Col,
	}
	if rec, ok := p.records[key]; ok {
		return rec
	}
	if p.records == nil {
		p.records = make(map[profileKey]*profileRecord)
	}
	rec := &profileRecord{owner: p, key: key, position: position}
	p.records[key] = rec
	return rec
}

// A zero address means that the callable's address is not available. Common
// callable types skip reflection because this code runs for every call.
func profileIdentity(c Callable) (uintptr, syntax.Position) {
	switch c := c.(type) {
	case *Function:
		return uintptr(unsafe.Pointer(c.funcode)), c.funcode.Pos
	case *Builtin:
		return reflect.ValueOf(c.fn).Pointer(), builtinPosition
	}
	var address uintptr
	if v := reflect.ValueOf(c); v.Type().Kind() == reflect.Pointer {
		address = v.Pointer()
	}
	return address, callablePosition(c)
}

func callablePosition(c Callable) syntax.Position {
	if c, ok := c.(callableWithPosition); ok {
		return c.Position()
	}
	return builtinPosition
}

var builtinPosition = syntax.MakePosition(&builtinFilename, 0, 0)

func (thread *Thread) recordCallPush(fr *frame) {
	// Close any span left open by a removed profile.
	if thread.Profile == nil && !thread.execProfileSpan.isOpen() {
		return
	}
	thread.recordCallPushImpl(fr)
}

func (thread *Thread) recordCallPushImpl(fr *frame) {
	now := nanotime()
	callerSpanWasOpen := thread.execProfileSpan.isOpen()
	thread.execProfileSpan.close(now)
	p := thread.Profile
	if p == nil {
		return
	}

	record := p.recordForCallable(fr.callable)
	record.activeCalls++
	fr.execProfileRecord = record
	fr.execProfileCallStart = now
	fr.callerProfileSpanWasOpen = callerSpanWasOpen
	thread.execProfileSpan.open(record, now)
}

func (thread *Thread) recordCallPop(fr *frame) {
	// The frame keeps this state after profile removal or a panic. In both
	// cases, activeCalls must still go down.
	if fr.execProfileRecord == nil {
		return
	}
	thread.recordCallPopImpl(fr)
}

func (thread *Thread) recordCallPopImpl(fr *frame) {
	now := nanotime()
	thread.execProfileSpan.close(now)

	record := fr.execProfileRecord
	record.activeCalls--
	if thread.Profile == record.owner {
		elapsed := time.Duration(now - fr.execProfileCallStart)
		record.completedCalls++
		if elapsed > record.maxTime {
			record.maxTime = elapsed
		}
		if record.activeCalls == 0 {
			record.cumulativeTime += elapsed
		}
	}

	// When Load runs Starlark again on the same thread, its caller must stay
	// paused. If the profile changed, do not resume a caller from the old one.
	if fr.callerProfileSpanWasOpen && len(thread.stack) > 1 {
		if caller := thread.frameAt(1); caller.execProfileRecord.owner == thread.Profile {
			thread.execProfileSpan.open(caller.execProfileRecord, now)
		}
	}
}

func (thread *Thread) recordPause() {
	if !thread.execProfileSpan.isOpen() {
		return
	}
	thread.recordPauseImpl()
}

func (thread *Thread) recordPauseImpl() {
	thread.execProfileSpan.close(nanotime())
}

func (thread *Thread) recordResume() {
	if thread.Profile == nil || thread.execProfileSpan.isOpen() {
		return
	}
	thread.recordResumeImpl()
}

func (thread *Thread) recordResumeImpl() {
	if fr := thread.frameAt(0); fr.execProfileRecord != nil && fr.execProfileRecord.owner == thread.Profile {
		thread.execProfileSpan.open(fr.execProfileRecord, nanotime())
	}
}

// An ExecProfileRecord reports the execution of one function.
type ExecProfileRecord struct {
	// Name is the function's Callable.Name. It is "<toplevel>" for a module's
	// top-level code.
	Name string

	// Position is where the function is defined. A function with no definition
	// position uses the special filename "<builtin>".
	Position syntax.Position

	// Calls is the number of calls that ended. This includes a normal return,
	// an error, or a panic.
	Calls int64

	// Self is the time when this function was the innermost function running
	// Starlark code. It does not include time in functions it calls or time
	// waiting for Load. It includes time in Print and OnMaxSteps because those
	// callbacks do not create frames.
	Self time.Duration

	// Cum is the total time from the start to the end of each outermost call to
	// this function. For recursive calls, it counts the full outer call once.
	// It includes time in called functions and time waiting for Load. The
	// top-level function's Cum is close to the full execution time.
	Cum time.Duration

	// Max is the longest time from the start to the end of one completed call.
	Max time.Duration
}

// Records returns one record for each function, sorted by decreasing Self.
func (p *ExecProfile) Records() []ExecProfileRecord {
	// Two built-ins can have the same Self, Name, and Position. Use their
	// addresses as the final sort value so map iteration order does not change
	// the result.
	sorted := make([]*profileRecord, 0, len(p.records))
	for _, rec := range p.records {
		sorted = append(sorted, rec)
	}
	slices.SortFunc(sorted, func(x, y *profileRecord) int {
		return cmp.Or(
			cmp.Compare(y.selfTime, x.selfTime),
			cmp.Compare(x.key.name, y.key.name),
			comparePosition(x.position, y.position),
			cmp.Compare(x.key.address, y.key.address),
		)
	})

	records := make([]ExecProfileRecord, len(sorted))
	for i, rec := range sorted {
		records[i] = ExecProfileRecord{
			Name:     rec.key.name,
			Position: rec.position,
			Calls:    rec.completedCalls,
			Self:     rec.selfTime,
			Cum:      rec.cumulativeTime,
			Max:      rec.maxTime,
		}
	}
	return records
}

// A syntax.Position stores a filename through a pointer. Compare the filename
// text so equal filenames are treated as equal.
func comparePosition(p, q syntax.Position) int {
	if pf, qf := p.Filename(), q.Filename(); pf != qf {
		return strings.Compare(pf, qf)
	}
	if p.Line != q.Line {
		return int(p.Line - q.Line)
	}
	return int(p.Col - q.Col)
}

// Merge combines q's records into p. Neither profile may be attached to a
// running thread.
func (p *ExecProfile) Merge(q *ExecProfile) {
	if q == nil || q == p {
		return
	}
	for key, qrec := range q.records {
		prec, ok := p.records[key]
		if !ok {
			if p.records == nil {
				p.records = make(map[profileKey]*profileRecord)
			}
			prec = &profileRecord{owner: p, key: key, position: qrec.position}
			p.records[key] = prec
		}
		prec.completedCalls += qrec.completedCalls
		prec.selfTime += qrec.selfTime
		prec.cumulativeTime += qrec.cumulativeTime
		if qrec.maxTime > prec.maxTime {
			prec.maxTime = qrec.maxTime
		}
	}
}

// WriteText writes a plain-text table to w. It writes a header, followed by
// one line for each record in the order returned by [ExecProfile.Records]:
//
//	calls       self        cum        max  function
//	 1042    41.52ms    41.52ms     2.10ms  sha256 (hash.star:12:1)
//	    3    12.08ms    93.11ms    40.07ms  build (build.star:7:1)
//	    1   830.00µs   140.20ms   140.20ms  <toplevel> (main.star:1:1)
//
// WriteText has no options. To change the columns, units, order, or filters,
// call [ExecProfile.Records] and format the records in the calling code.
func (p *ExecProfile) WriteText(w io.Writer) error {
	// Build the full table before making one write to w.
	buf := new(strings.Builder)
	fmt.Fprintf(buf, "%8s %10s %10s %10s  %s\n", "calls", "self", "cum", "max", "function")
	for _, rec := range p.Records() {
		fmt.Fprintf(buf, "%8d %10s %10s %10s  %s (%s)\n",
			rec.Calls,
			profileDuration(rec.Self),
			profileDuration(rec.Cum),
			profileDuration(rec.Max),
			rec.Name,
			rec.Position)
	}
	_, err := io.WriteString(w, buf.String())
	return err
}

func profileDuration(d time.Duration) string {
	switch {
	case d < time.Microsecond:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	case d < time.Millisecond:
		return fmt.Sprintf("%.2fµs", float64(d)/float64(time.Microsecond))
	case d < time.Second:
		return fmt.Sprintf("%.2fms", float64(d)/float64(time.Millisecond))
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
