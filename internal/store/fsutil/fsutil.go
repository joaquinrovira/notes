package fsutil

// Naive implementation Lock and Unlock of [billy.File].
// Just have to satisfy the interface. We're not really using locks.
type Locker struct{}

func (f Locker) Lock() error {
	return nil
}
func (f Locker) Unlock() error {
	return nil
}
