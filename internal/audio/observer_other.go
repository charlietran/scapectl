//go:build !darwin

package audio

// Start is a no-op on platforms without the CoreAudio observer.
func Start() (stop func(), err error) {
	return func() {}, nil
}
