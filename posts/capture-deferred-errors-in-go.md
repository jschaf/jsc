+++
slug = "capture-deferred-errors-in-go"
date = 2025-01-02
visibility = "published"
+++

# Capture deferred errors in Go

One of my obsessions is to capture errors diligently. Most often, I satiate my
desire with the divisive incantation `if err != nil { return fmt.Errorf(...) }`.
Errors in defer statements require a more delicate touch. Opening a file demands
a matching close to avoid leaking file descriptors. For example, we might open a
file and write interesting data:

```go {description="motivating example"}
func populateFile() error
	f, err := os.Open("foo.txt", os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close() // BAD: ignored error <HL>
	
	err = writeInterestingData(f)
	if err != nil {
		return fmt.Errorf("write data: %w", err)
	}
	return nil
}
```

Problematically, we ignore the error when closing the file. The usual remedy is
to defer an immediately invoked function expression ([IIFE] for short, a term our
JavaScript friends might recognize). The IIFE overwrites the colloquially-named
`err` return argument.

[IIFE]: https://developer.mozilla.org/en-US/docs/Glossary/IIFE

```go {description="overwriting the error in defer"}
func populateFile() (err error)
	f, err := os.Open("foo.txt", os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer func() {                                       // <HL>
		if closeErr := f.Close(); err != nil {           // <HL>
			// BAD: overwrites existing error               <HL>
			err = fmt.Errorf("close file: %w", closeErr) // <HL>
		}                                                // <HL>
	}()                                                  // <HL>

	err = writeInterestingData(f)
	if err != nil {
		return fmt.Errorf("write data: %w", err)
	}
	return nil
}
```

Alas, this solution brings its own set of problems. If `f.Close` errors, we
overwrite the error from `writeInterestingData`. We need to combine the errors.
Before reaching to Uber's [multierr] package, we'll lean on [`errors.Join`] to
combine multiple errors, introduced by Go 1.20.

[`errors.Join`]: https://pkg.go.dev/errors#Join

```go {description="correct and verbose"}
func populateFile() (err error)
	f, err := os.Open("foo.txt", os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer func() {                                                         // <HL>
		if closeErr := f.Close(); err != nil {                             // <HL>
			err = errors.Join(err, fmt.Errorf("close file: %w", closeErr)) // <HL>
		}                                                                  // <HL>
	}()                                                                    // <HL>

	err = writeInterestingData(f)
	if err != nil {
		return fmt.Errorf("write data: %w", err)
	}
	return nil
}
```

The solution does not please the eyes and requires choosing a name other than
`err`, like `closeErr` for the `f.Close` error.

[multierr]: https://github.com/uber-go/multierr

## Captured by Thanos

Inspired by Thanos' [coding style guide], we'll simplify the unwieldy anonymous
function. Thanos defines [`runutil.CloseWithErrCapture`], which calls Close and combines
the error with an existing named error.

[coding style guide]: https://thanos.io/tip/contributing/coding-style-guide.md/#defers-dont-forget-to-check-returned-errors
[`runutil.CloseWithErrCapture`]: https://github.com/thanos-io/thanos/blob/ca40906c83d94cfcbe4bcc181a286663aeb268d5/pkg/runutil/runutil.go#L156

```go {name="runutil.go" description="helper functions from thanos"}
// CloseWithErrCapture closes closer, wraps any error with message from
// fmt and args, and stores this in err.
func CloseWithErrCapture(err *error, c io.Closer, format string, a ...any) {
	merr := errutil.MultiError{}
	merr.Add(*err)
	merr.Add(errors.Wrapf(c.Close(), format, a...))
	*err = merr.Err()
}
```

Armed by Thanos, we'll replace the anonymous function with `CloseWithErrCapture`.

```go {description="correct and less verbose"}
func populateFile() (err error)
	f, err := os.Open("foo.txt", os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer runutil.CloseWithErrCapture(&err, f, "close file") // <HL>

	err = writeInterestingData(f)
	if err != nil {
		return fmt.Errorf("write data: %w", err)
	}
	return nil
}
```

## An err of simplicity

We'll cull half the complexity from `CloseWithErrCapture` with a snap. While
we're at it, we'll generalize the pattern to any error-returning function named
`errs.Capture`. In our 300 kLOC monorepo, we use `errs.Capture` 554 times. Only
60% of the calls are for `io.Closer.Close`. The remaining calls are cleanup
functions, like `Flush`, `Shutdown`, and functions requiring context, like
`pgx.Conn.Close(ctx)`.

```go {description="errs.Capture revealed"}
package errs

import (
	"errors"
	"fmt"
)

// Capture runs errFunc and assigns the error, if any, to *errPtr.
// Preserves the original error by wrapping with errors.Join if
// errFunc returns a non-nil error.
func Capture(errPtr *error, errFunc func() error, msg string) {
	err := errFunc()
	if err == nil {
		return
	}
	*errPtr = errors.Join(*errPtr, fmt.Errorf("%s: %w", msg, err))
}
```

Instead of using an `io.Closer` interface, we'll pass the function method at the
call-site.

```go {description="corrected motivating example"}
func populateFile() (err error)
	f, err := os.Open("foo.txt", os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer errs.Capture(&err, f.Close, "close file") // <HL>

	err = writeInterestingData(f)
	if err != nil {
		return fmt.Errorf("write data: %w", err)
	}
	return nil
}
```

Our `errs.Capture` outshines on `runutil.CloseWithErrCapture` in three ways.
First, the name is shorter, more direct, and avoids the [bad package name]
`runutil`. Second, by generalizing to any error-returning function, we've moved
`Close` out of the implementation and to the call site, removing a layer of
indirection. Third, the function stands alone, implemented solely in the
standard library.

[bad package name]: https://go.dev/blog/package-names#bad-package-names

## Extensions

We've considered a few similar capture functions but only implemented
`errs.CaptureT` to capture errors in a test. We call the testing variant 125
times in our 300 kLOC monorepo.

```go {description="capture errors in tests"}
package errs

// testingTB is a subset of *testing.T and *testing.B methods.
type testingTB interface {
	Helper()
	Errorf(format string, args ...interface{})
}

// CaptureT call t.Errorf if errFunc returns an error with a message.
func CaptureT(t testingTB, errFunc func() error, msg string) {
	t.Helper()
	if err := errFunc(); err != nil {
		t.Errorf("%s: %s", msg, err)
	}
}
```

Extensions we haven't implemented since the [utility and ubiquity] is low:

[utility and ubiquity]: https://github.com/google/guava/wiki/PhilosophyExplained#when-in-doubt

- `errs.CaptureContext` for capturing errors from functions that take a
  context.Context and return an error. It's not much more code to use errs.Capture
  with an anonymous function. Only 25 of the 554 calls to `errs.Capture` need
  a context.

- `errs.CaptureLog` to log the error. We only log deferred errors a handful of
  times.

- `errs.Capture` support for formatted errors. We only used format support a
  handful of times. The pain of calling fmt.Sprintf didn't outweigh the (minor)
  complexity of supporting formatted arguments.
