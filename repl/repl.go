// The repl package provides a read/eval/print loop for Skylark.
//
// It supports readline-style command editing,
// and interrupts through Control-C.
//
// If an input line can be parsed as an expression,
// the REPL parses and evaluates it and prints its result.
// Otherwise the REPL reads lines until a blank line,
// then tries again to parse the multi-line input as an
// expression. If the input still cannot be parsed as an expression,
// the REPL parses and executes it as a file (a list of statements),
// for side effects.
package repl

// TODO(adonovan):
//
// - Unparenthesized tuples are not parsed as a single expression:
//     >>> (1, 2)
//     (1, 2)
//     >>> 1, 2
//     ...
//     >>>
//   This is not necessarily a bug.

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/chzyer/readline"
	"github.com/google/skylark"
	"github.com/google/skylark/syntax"
)

var interrupted = make(chan os.Signal, 1)

// REPL executes a read, eval, print loop.
//
// Before evaluating each expression, it sets the Skylark thread local
// variable named "context" to a context.Context that is cancelled by a
// SIGINT (Control-C). Client-supplied global functions may use this
// context to make long-running operations interruptable.
//
func REPL(thread *skylark.Thread, globals skylark.StringDict) {
	signal.Notify(interrupted, os.Interrupt)
	defer signal.Stop(interrupted)

	rl, err := readline.New(">>> ")
	if err != nil {
		PrintError(err)
		return
	}
	defer rl.Close()
	for {
		if err := rep(rl, thread, globals); err != nil {
			if err == readline.ErrInterrupt {
				fmt.Println(err)
				continue
			}
			break
		}
	}
	fmt.Println()
}

// rep reads, evaluates, and prints one item.
//
// It returns an error (possibly readline.ErrInterrupt)
// only if readline failed. Skylark errors are printed.
func rep(rl *readline.Instance, thread *skylark.Thread, globals skylark.StringDict) error {
	// Each item gets its own context,
	// which is cancelled by a SIGINT.
	//
	// Note: during Readline calls, Control-C causes Readline to return
	// ErrInterrupt but does not generate a SIGINT.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-interrupted:
			cancel()
		case <-ctx.Done():
		}
	}()

	thread.SetLocal("context", ctx)

	rl.SetPrompt(">>> ")
	line, err := rl.Readline()
	if err != nil {
		return err // may be ErrInterrupt
	}

	if l := strings.TrimSpace(line); l == "" || l[0] == '#' {
		return nil // blank or comment
	}

	// If the line contains a well-formed expression, evaluate it.
	if _, err := syntax.ParseExpr("<stdin>", line, 0); err == nil {
		if v, err := skylark.Eval(thread, "<stdin>", line, globals); err != nil {
			PrintError(err)
		} else if v != skylark.None {
			fmt.Println(v)
		}
		return nil
	}

	// If the input so far is a single load or assignment statement,
	// execute it without waiting for a blank line.
	if f, err := syntax.Parse("<stdin>", line, 0); err == nil && len(f.Stmts) == 1 {
		switch f.Stmts[0].(type) {
		case *syntax.AssignStmt, *syntax.LoadStmt:
			// Execute it as a file.
			if err := execFileNoFreeze(thread, line, globals); err != nil {
				PrintError(err)
			}
			return nil
		}
	}

	// Otherwise assume it is the first of several
	// comprising a file, followed by a blank line.
	var buf bytes.Buffer
	fmt.Fprintln(&buf, line)
	for {
		rl.SetPrompt("... ")
		line, err := rl.Readline()
		if err != nil {
			return err // may be ErrInterrupt
		}
		if l := strings.TrimSpace(line); l == "" {
			break // blank
		}
		fmt.Fprintln(&buf, line)
	}
	text := buf.Bytes()

	// Try parsing it once more as an expression,
	// such as a call spread over several lines:
	//   f(
	//     1,
	//     2
	//   )
	if _, err := syntax.ParseExpr("<stdin>", text, 0); err == nil {
		if v, err := skylark.Eval(thread, "<stdin>", text, globals); err != nil {
			PrintError(err)
		} else if v != skylark.None {
			fmt.Println(v)
		}
		return nil
	}

	// Execute it as a file.
	if err := execFileNoFreeze(thread, text, globals); err != nil {
		PrintError(err)
	}

	return nil
}

// execFileNoFreeze is skylark.ExecFile without globals.Freeze().
func execFileNoFreeze(thread *skylark.Thread, src interface{}, globals skylark.StringDict) error {
	_, prog, err := skylark.SourceProgram("<stdin>", src, globals.Has)
	if err != nil {
		return err
	}

	res, err := prog.Init(thread, globals)

	// The global names from the previous call become
	// the predeclared names of this call.

	// Copy globals back to the caller's map.
	// If execution failed, some globals may be undefined.
	for k, v := range res {
		globals[k] = v
	}

	return err
}

// PrintError prints the error to stderr,
// or its backtrace if it is a Skylark evaluation error.
func PrintError(err error) {
	if evalErr, ok := err.(*skylark.EvalError); ok {
		fmt.Fprintln(os.Stderr, evalErr.Backtrace())
	} else {
		fmt.Fprintln(os.Stderr, err)
	}
}

// MakeLoad returns a simple sequential implementation of module loading
// suitable for use in the REPL.
// Each function returned by MakeLoad accesses a distinct private cache.
func MakeLoad() func(thread *skylark.Thread, module string) (skylark.StringDict, error) {
	type entry struct {
		globals skylark.StringDict
		err     error
	}

	var cache = make(map[string]*entry)

	return func(thread *skylark.Thread, module string) (skylark.StringDict, error) {
		e, ok := cache[module]
		if e == nil {
			if ok {
				// request for package whose loading is in progress
				return nil, fmt.Errorf("cycle in load graph")
			}

			// Add a placeholder to indicate "load in progress".
			cache[module] = nil

			// Load it.
			thread := &skylark.Thread{Load: thread.Load}
			globals, err := skylark.ExecFile(thread, module, nil, nil)
			e = &entry{globals, err}

			// Update the cache.
			cache[module] = e
		}
		return e.globals, e.err
	}
}
