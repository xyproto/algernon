package symwalk

// This files is based on stretchr/powerwalk, which is also MIT licensed (credits given in the LICENSE file)

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// DefaultConcurrentWalks is the default number of files that will be walked at the
// same time when the Walk function is called.
// To use a value other than this one, use the WalkLimit function.
const DefaultConcurrentWalks int = 100

// Walk walks the file tree rooted at root, calling walkFn for each file or
// directory in the tree, including root. All errors that arise visiting files
// and directories are filtered by walkFn. The output is non-deterministic.
// WalkLimit does not follow symbolic links.
//
// For each file and directory encountered, Walk will trigger a new Go routine
// allowing you to handle each item concurrently.  A maximum of DefaultConcurrentWalks
// walkFns will be called at any one time.
func Walk(root string, walkFn filepath.WalkFunc) error {
	return WalkLimit(root, walkFn, DefaultConcurrentWalks)
}

// WalkLimit walks the file tree rooted at root, calling walkFn for each file or
// directory in the tree, including root. All errors that arise visiting files
// and directories are filtered by walkFn. The output is non-deterministic.
// WalkLimit does not follow symbolic links.
//
// For each file and directory encountered, Walk will trigger a new Go routine
// allowing you to handle each item concurrently.  A maximum of limit walkFns will
// be called at any one time.
func WalkLimit(root string, walkFn filepath.WalkFunc, limit int) error {
	if limit < 1 {
		panic("powerwalk: limit must be greater than zero.")
	}

	var (
		files    = make(chan *walkArgs)
		done     = make(chan struct{})
		doneOnce sync.Once
		errMu    sync.Mutex
		walkErr  error
	)
	// setErr records the first error from a walkFn invocation and signals
	// every goroutine to stop. close(done) is guarded by sync.Once so that
	// concurrent errors cannot panic the program.
	setErr := func(err error) {
		errMu.Lock()
		if walkErr == nil {
			walkErr = err
		}
		errMu.Unlock()
		doneOnce.Do(func() { close(done) })
	}
	getErr := func() error {
		errMu.Lock()
		defer errMu.Unlock()
		return walkErr
	}

	// Workers consume files until the channel is closed (normal completion)
	// or until done is closed (an error was recorded by some other worker).
	var workersWg sync.WaitGroup
	for i := 0; i < limit; i++ {
		workersWg.Add(1)
		go func() {
			defer workersWg.Done()
			for {
				select {
				case <-done:
					return
				case file, ok := <-files:
					if !ok {
						return
					}
					if err := walkFn(file.path, file.info, file.err); err != nil {
						setErr(err)
						return
					}
				}
			}
		}()
	}

	// Walker drives filepathWalk. Each visited entry is either dispatched to
	// a worker or, if an error has already been observed, the walk aborts.
	var walkerWg sync.WaitGroup
	walkerWg.Add(1)
	go func() {
		defer walkerWg.Done()
		defer close(files)
		filepathWalk(root, func(p string, info os.FileInfo, err error) error {
			select {
			case <-done:
				return errors.New("kill received while walking")
			case files <- &walkArgs{path: p, info: info, err: err}:
				return nil
			}
		})
	}()

	walkerWg.Wait()
	workersWg.Wait()
	return getErr()
}

// walkArgs holds the arguments that were passed to the Walk or WalkLimit
// functions.
type walkArgs struct {
	info os.FileInfo
	err  error
	path string
}
