package paths

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/jschaf/jsc/pkg/errs"
	"github.com/jschaf/jsc/pkg/files"
	"github.com/karrick/godirwalk"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// WalkUp traverses up directory tree until it finds an ancestor directory that
// contains dirToFind. WalkUp checks the current directory and then
func WalkUp(dirToFind string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working dir: %s", err)
	}

	for dir != string(os.PathSeparator) {
		p := filepath.Join(dir, dirToFind)

		if stat, err := os.Stat(p); err != nil {
			if !os.IsNotExist(err) {
				return "", fmt.Errorf("stat dirToFind %s: %w", p, err)
			}
		} else if stat.IsDir() {
			return dir, nil
		}

		dir = filepath.Dir(dir)
	}
	return "", fmt.Errorf("dir not found in directory tree starting from %s", dir)
}

// WalkConcurrent walks dir, recursively calling walkFunc on each entry.
func WalkConcurrent(dir string, maxParallel int, walkFunc godirwalk.WalkFunc) error {
	sem := semaphore.NewWeighted(int64(maxParallel))
	ctx := context.Background()
	g := &errgroup.Group{}

	callback := func(path string, dirent *godirwalk.Dirent) error {
		if err := sem.Acquire(ctx, 1); err != nil {
			return fmt.Errorf("walk concurrent acquire semaphore: %w", err)
		}
		g.Go(func() error {
			defer sem.Release(1)
			return walkFunc(path, dirent)
		})
		return nil
	}
	err := godirwalk.Walk(dir, &godirwalk.Options{Unsorted: true, Callback: callback})
	if err != nil {
		return fmt.Errorf("walk concurrent walk error: %w", err)
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("walk concurrent wait err group: %w", err)
	}
	return nil
}

type WalkCollectFunc[T any] func(path string, directoryEntry fs.DirEntry) ([]T, error)

// WalkCollect walks dir, recursively calling walkFunc on each entry collecting
// a slice of T from each walkFunc.
func WalkCollect[T any](dir string, walkFunc WalkCollectFunc[T]) ([]T, error) {
	sem := semaphore.NewWeighted(int64(runtime.NumCPU()))
	eg, ctx := errgroup.WithContext(context.Background())
	mu := sync.Mutex{}
	vals := make([]T, 0, 4)

	err := filepath.WalkDir(dir, func(path string, dirent fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if err := sem.Acquire(ctx, 1); err != nil {
			return fmt.Errorf("walk collect acquire semaphore: %w", err)
		}
		eg.Go(func() error {
			defer sem.Release(1)
			vs, err := walkFunc(path, dirent)
			if err != nil {
				return err
			}
			if len(vs) > 0 {
				mu.Lock()
				vals = append(vals, vs...)
				mu.Unlock()
			}
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk collect walk error: %w", err)
	}
	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("walk collect wait err group: %w", err)
	}
	return vals, nil
}

// Copy the contents of the src file to dest. Any existing file will be
// overwritten and will not copy file attributes.
func Copy(dest, src string) (mErr error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer errs.Capture(&mErr, in.Close, "")

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("mkdir to copy to dest %q: %w", dest, err)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer errs.Capture(&mErr, out.Close, "")

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return nil
}

// CopyLazy copies the contents of the src file to dest only if the contents
// are different. If the files are different, the existing file will be
// overwritten and will not copy file attributes. Returns true if the file was
// the same, otherwise false.
func CopyLazy(dest, src string) (b bool, mErr error) {
	if isSame, err := files.IsSameBytes(src, dest); errors.Is(err, os.ErrNotExist) {
		// Ok for the file not to exist.
	} else if err != nil {
		return false, fmt.Errorf("check if same file before copy: %w", err)
	} else if isSame {
		return true, nil
	}
	err := Copy(dest, src)
	return false, err
}
