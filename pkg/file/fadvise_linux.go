package file

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// markNotNeeded marks a function as 'not needed'.
// Interactions with large files can end up with a large page cache. This isn't really a big deal as the OS can reclaim
// this under memory pressure. However, Kubernetes counts page cache usage against the container memory usage.
// This leads to bloating up memory usage if we are just copying large files around.
func markNotNeeded(in *os.File) error {
	err := unix.Fadvise(int(in.Fd()), 0, 0, unix.FADV_DONTNEED)
	if err != nil {
		return fmt.Errorf("failed to mark file FADV_DONTNEED: %v", err)
	}
	return nil
}
