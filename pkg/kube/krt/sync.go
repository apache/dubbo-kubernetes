package krt

type Syncer interface {
	WaitUntilSynced(stop <-chan struct{}) bool
}
