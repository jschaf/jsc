+++
slug = "delay-context-cancelation-in-go"
date = 2025-01-12
visibility = "published"
+++

# Delay context cancellation in Go

The Go standard library defines [`context.Context`] to propagate deadlines,
cancellation signals, and request-scoped values. For a server, the main function
usually creates a top-level context that is canceled by an interrupt signal.
Today, we'll explore the interaction of context cancellation with batch
workloads (as opposed to request-response workloads), like exporting traces.

```go {description="cancel context on SIGINT"}
package main

import (
	"context"
	"os"
	"os/signal"
)

func runMain() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	
	tracer := NewTracer(ctx)
	trace.SetDefault(tracer)
	defer tracer.Flush()
  
	return runMyServer(ctx)
}
```

[`context.Context`]: https://pkg.go.dev/context

Once a context is canceled, all requests using that context fail. Batched
workloads, like logs and traces, will fail to send queued data as the server
shuts down. We'll have an observability gap during the server shutdown.

## Upstream issue

The Go issue, [context: add WithoutCancel], discusses the nuances and bemoans
conflating cancellation and context values. The issue took just under three
years, July 2020 to May 2023, to land in the Go standard library.
The two reasons in the linked issue for removing cancellation are:

1. Processing rollbacks or other cleanup.
2. Observability tasks to run after the context is canceled.

The second reason _precisely_ describes our use case of exporting traces.

[context: add WithoutCancel]: https://github.com/golang/go/issues/40221

## WithDelayedCancel

With the new Go additions, we can create a derived context that cancels after
the parent context is done. For a dash of mechanical sympathy, we'll use the new
[context.AfterFunc] to avoid starting a goroutine until the context is done.

[context.AfterFunc]: https://pkg.go.dev/context#AfterFunc

```go {description="derived context canceled after the parent"}

package contexts

import (
	"context"
	"time"
)

// WithDelayedCancel returns a new context that cancels
// after the parent is done plus a delay. Useful to flush
// traces without dropping all traces because the context
// is canceled.
func WithDelayedCancel(parent context.Context, delay time.Duration) context.Context {
	child, childCancel := context.WithCancel(context.WithoutCancel(parent))
	context.AfterFunc(parent, func() {
		time.Sleep(delay)
		childCancel()
	})
	return child
}
```

We use WithDelayedCancel to allow the tracer a grace period to flush traces.

```go {description="cancel context on SIGINT"}
package main

import (
	"context"
	"os"
	"os/signal"
	"time"
)

func runMain() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	flushGracePeriod := 5 * time.Second                                    // <HL>
	tracer := NewTracer(contexts.WithDelayedCancel(ctx, flushGracePeriod)) // <HL>
	trace.SetDefault(tracer)
	defer tracer.Flush()

	return runMyServer(ctx)
}
```
