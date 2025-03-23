+++
slug = "typescript-semaphore"
date = 2020-10-15
visibility = "published"
bib_paths = ["/ref.bib"]
+++

# Creating a semaphore in TypeScript

:toc:

I recently needed a semaphore in our Node.js backend service to run database
queries concurrently up to a certain threshold. It was a fun diversion from the
more regular work of copying JSON from one destination to another.

## What is a semaphore?

Counting [semaphores] are used to "control the number of activities that can
access a certain resource or perform a given action at the same time."
[^@goetz2006java]

[semaphores]: https://en.wikipedia.org/wiki/Semaphore_(programming)

## Memory visibility and mutual exclusion

The two foundational concepts of concurrency relevant to semaphores are memory
visibility and mutual exclusion, but we’ll only need to implement mutual
exclusion. To see why we don’t need to implement memory visibility, let’s
consider the definition of memory visibility: a system has memory visibility if
when one thread modifies a shared variable, the change is visible to other
threads. The Node.js runtime guarantees memory visibility by virtue of running
application code in a single thread and dispatching asynchronous calls to four
worker threads distributed by libuv. That leaves mutual exclusion, which we’ll
implement in the semaphore using promises. The flavor of mutual exclusion
relevant to semaphores is ensuring no more than *n* tasks are running
concurrently.

## API design

Let’s start with the desired API. We’ll create a weighted semaphore, meaning
callers can acquire and release arbitrary weighted leases into the semaphore. A
weighted semaphore supports patterns like weighting expensive database queries
more heavily than cheaper queries.

- `Semaphore.newWeighted(n: number)` - a static factory [^side:static-factories]
  that creates a new Semaphore that allows up to n weight. 
- `Semaphore.acquire(n: number) => Promise<void>` - a method that blocks until at
  least n capacity is available in the semaphore.
- `Semaphore.release(n: number) => void` - a non-blocking method that releases n
  capacity back to the semaphore.
  
::: footnote side:static-factories
I prefer static factories to constructors because static factories can have 
descriptive names.
:::
  
The API is functionally equivalent to the standard library adjacent
[golang.org/x/sync/semaphore][go_sem] Golang package. Here’s how’d we’d use the 
above API to limit the number of concurrent database connections.

[go_sem]: https://pkg.go.dev/golang.org/x/sync/semaphore

CONTINUE_READING
  
```ts
const db = createDbConnection();
const createEmployee = (employee): Promise<void> => {
  return db.exec(
    'INSERT INTO employee (id, name) VALUES ($1, $2);',
    [employee.id, employee.name],
  );
};

const sem = Semphore.newWeighted(10);
// Insert up to 10 employees concurrently.
for (const emp of employees) {
  await sem.acquire(1);
  // Floating promise; we can't await here. See below.
  createEmployee(emp)
    .finally(() => sem.release(1));
}
```

The usage looks pretty normal except for the floating promise created by
`createEmployee(emp)`. In normal code, we should use await
`createEmployee(emp)`, because createEmployee returns a promise. However, if we
do use await, the loop will process employees sequentially, instead of
processing 10 employees at a time. Right now, this code will throw an unhandled
promise rejection warning because if createEmployee fails, the application code
will catch the error.

The fix is to handle errors in the floating promise separately and to error out
of the loop if any errors exist.

```ts
const errs: Array<Error> = [];
for (const emp of employees) {
  await sem.acquire(1);
  if (errs.length > 0) {
    sem.release(1);
    break;
  }
  createEmployee(emp)
    .catch((e) => errs.push(e))
    .finally(() => sem.release(1));
}

// We also have to check for errors because any promise 
// in the last batch might have errored.
if (errs.length > 0) {
  throw errs[0];
}
```

There’s still a bug in the above code snippet. What happens if we create
`Semaphore.newWeighted(100)` and there’s only 5 employees? The for-loop will
process all employees immediately without waiting on `Semaphore.acquire` because
there’s enough capacity to run through all employees concurrently. That means
we’ll exit the for-loop while the createEmployee promises are still running.
After exiting the loop, we’ll check `errs.length` before all promises are
resolved meaning we might ignore errors.

We need some way to wait for all createEmployee promises to resolve. We can do
that indirectly by trying to acquire the entire capacity of a Semaphore. For our
semaphore with 100 capacity, that would look like `sem.acquire(100)`. We cannot
acquire all the capacity until all pending and currently running promises
release capacity back to the semaphore. This is a really useful concept, so I
added a new method, `Semaphore.wait` The advantage of a separate wait method is
that you don’t need to specify the weight. We can fix our example with:

```ts
const errs: Array<Error> = [];
for (const emp of employees) {
  await sem.acquire(1);
  if (errs.length > 0) {
    sem.release(1);
    break;
  }
  createEmployee(emp)
    .catch((e) => errs.push(e))
    .finally(() => sem.release(1));
}

await sem.wait();

if (errs.length > 0) {
  throw errs[0];
}
```

## Implementation

We’ll walk through the implementation in steps. First up, the static factory,
`Semaphore.newWeighted` to initialize the following shared state: 

- `pending`: An array of two-tuples containing all the weight and promise of all
acquire calls. The Semaphore class resolves the promise once it admits the
pending entry.

- `running`: The currently used capacity of the Semaphore.

```ts
class Semaphore {
  // Each tuple is [weight, promise]. The promise is resolved once 
  // the semaphore has enough capacity, where:
  //   weight <= (this.maxWeight - this.running).
  private readonly pending: Array<[number, SettablePromise<void>]> = [];
  // The currently used capacity of the semaphore, in 
  // range [0, maxWeight].
  private running: number = 0;
  private constructor(private readonly maxWeight: number) {}

  /**
   * Creates a new weighted semaphore with the given maximum 
   * combined weight for concurrent access.
   */
  static newWeighted(n: number): Semaphore {
    return new Semaphore(n);
  }
}
```

Next up, we’ll walk through the implementation of `acquire`.

- If the semaphore has enough capacity for the requested weight, immediately
  resolve the promise.
- Otherwise, create a new promise that will be resolved by a future `release`
  call.

```ts
class Semaphore {
  /**
   * Acquires the semaphore with a weight of n, blocking until at 
   * least n capacity is available.
   */
  async acquire(n: number): Promise<void> {
    if (this.maxWeight >= this.running + n) {
      // If we can run this task now, do it immediately.
      this.running += n;
      return;
    }
    // Create a promise that a future release() call can resolve once
    // there's enough capacity. Since we checked n <= this.maxWeight,
    // this promise will eventually be resolved.
    const sp = SettablePromise.create<void>();
    this.pending.push([n, sp]);
    await sp;
    // We'll increment this.running in release().
  }
}
```

Next up is `release`. Release for a weighted semaphore is a bit more complicated
than a semaphore with uniform weights. Releasing a heavily-weighted semaphore
like `release(10)` might allow 10 separate semaphores with `acquire(1)` to
start. Imagine a scenario where we use different weights:


```ts
// Example: releasing a heavily weighted semaphore to trigger many 
// lightly weighted semaphores.

const sem = Semaphore.newWeighted(10);
sem.acquire(4).then(doThing).finally(() => sem.release(4));
sem.acquire(6).then(doThing).finally(() => sem.release(6));
// All capacity used at this point. The following semaphores are
// blocked until one of the above semaphore releases capacity.
sem.acquire(1).then(doCheaperThing).finally(() => sem.release(1));
sem.acquire(1).then(doCheaperThing).finally(() => sem.release(1));
sem.acquire(1).then(doCheaperThing).finally(() => sem.release(1));

// Once either the 4-weighted or 6-weighted task resolves, it should
// start the three 1-weight tasks above.
```

To implement `release`, first free the capacity that the current semaphore
holds. Second, continually loop over the pending queue and resolving promises
from waiting `acquire` calls. Break from the loop once there's not enough
capacity to run the next pending `acquire` call. Release implements a [noop
scheduler]. Pending `acquire` calls are resolved in the order that the calls
arrive. Since we're implementing a weighted semaphore we could implement smarter
scheduling algorithms. A non-weighted semaphore, or equivalently a Semaphore with
constant weights, would not benefit from another scheduler because there's no
way to determine the relative priority of incoming `acquire` calls.

[noop scheduler]: https://en.wikipedia.org/wiki/Noop_scheduler


```ts
class Semaphore{
  // Releases the semaphore with a weight of n.
  // Should be called in a try-finally block or Promise.finally
  // function or else you risk deadlock.
  release(n: number): void {
    this.running -= n;
    while (this.pending.length > 0) {
      const [nextWeight, nextPromise] = this.pending[0];
      if (this.running + nextWeight > this.maxWeight) {
        // Not enough capacity, let the next release try again.
        break;
      }
      this.pending.shift();
      this.running += nextWeight;
      // resolve the SettablePromise<void> of an acquire() call
      nextPromise.set();
    }
  }
}
```

Finally, let’s implement `wait`, which is mercifully simple. By trying to acquire the `maxWeight`,
`wait` guarantees that it will only start once all `pending` tasks `release` their capacity.

```ts
class Semaphore {
  // Waits for all pending operations to finish. Resets the 
  // semaphore to the initial capacity after all pending operations
  // finish.
  async wait(): Promise<void> {
    await this.acquire(this.maxWeight);
    this.release(this.maxWeight);
  }
}
```

## A safe wrapper with mapLimit

As a concrete example, we'll implement `mapLimit` from the widely used [async]
library using our weighted semaphore. `mapLimit` maps an asynchronous function
over an iterable returning a promise containing the array of results. the result
of applying the map function on $i^{\mathrm{th}}$ entry in the iterable is
stored in the $i^{\mathrm{th}}$ entry of the returned array.

[async]: https://caolan.github.io/async/v3/mapLimit.js.html

```ts
const mapLimit = async <T, R>(
  iter: Iterable<T>,
  fn: (t: T) => Promise<R>,
  limit: number,
): Promise<Array<R>> => {
  const sem = Semaphore.newWeighted(limit);
  const defaultSize = 16;
  const resolved: Array<R> = Array(defaultSize).fill(null);
  const rejected: Array<Error> = [];
  let idx = 0;
  for (const val of iter) {
    await sem.acquire(1);
    // If any fn() call rejected, error immediately to avoid running
    // more fn's.
    if (rejected.length > 0) {
      sem.release(1); // released so sem.wait() finishes below
      break;
    }
    const i = idx;
    fn(val)
      .then((r) => (resolved[i] = r))
      .catch((err) => rejected.push(err))
      .finally(() => sem.release(1));
    idx++;
  }

  await sem.wait();
  if (rejected.length > 0) {
    throw rejected[0];
  }
  return resolved;
};
```



I strongly recommend using wrappers around semaphores; it's very easy to
introduce subtle concurrency bugs. While I was writing the library I wrote two
bugs that were caught in code review.

::: preview https://en.wikipedia.org/wiki/Semaphore_(programming)
Semaphore (programming)

In computer science, a semaphore is a variable or abstract data type used to
control access to a common resource by multiple processes in a concurrent system
such as a multitasking operating system. A semaphore is simply a variable. This
variable is used to solve critical section problems and to achieve process
synchronization in the multiprocessing environment. A trivial semaphore is a
plain variable that is changed (for example, incremented or decremented, or
toggled) depending on programmer-defined conditions.

A useful way to think of a semaphore as used in the real-world system is as a
record of how many units of a particular resource are available, coupled with
operations to adjust that record *safely* (i.e., to avoid race conditions) as
units are acquired or become free, and, if necessary, wait until a unit of the
resource becomes available.
:::

