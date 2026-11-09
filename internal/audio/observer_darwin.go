package audio

import (
	"bytes"
	"fmt"
	"log"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	systemObject = 1
	globalScope  = 0x676c6f62 // glob
	nameSelector = 0x6c6e616d // lnam
	utf8Encoding = 0x08000100
)

type propertyAddress struct {
	selector uint32
	scope    uint32
	element  uint32
}

var defaultAddresses = [...]propertyAddress{
	{selector: 0x64496e20, scope: globalScope}, // dIn
	{selector: 0x644f7574, scope: globalScope}, // dOut
	{selector: 0x734f7574, scope: globalScope}, // sOut
}

var native struct {
	once     sync.Once
	err      error
	callback uintptr
	get      func(uint32, *propertyAddress, uint32, uintptr, *uint32, unsafe.Pointer) int32
	add      func(uint32, *propertyAddress, uintptr, uintptr) int32
	remove   func(uint32, *propertyAddress, uintptr, uintptr) int32
	length   func(uintptr) int64
	cstring  func(uintptr, []byte, int64, uint32) bool
	release  func(uintptr)
}

var listeners = struct {
	sync.Mutex
	next uintptr
	all  map[uintptr]*observer
}{all: make(map[uintptr]*observer)}

func loadNative() error {
	native.once.Do(func() {
		type symbol struct {
			name   string
			target any
		}
		for _, library := range []struct {
			path    string
			symbols []symbol
		}{
			{
				path: "/System/Library/Frameworks/CoreAudio.framework/CoreAudio",
				symbols: []symbol{
					{"AudioObjectGetPropertyData", &native.get},
					{"AudioObjectAddPropertyListener", &native.add},
					{"AudioObjectRemovePropertyListener", &native.remove},
				},
			},
			{
				path: "/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation",
				symbols: []symbol{
					{"CFStringGetLength", &native.length},
					{"CFStringGetCString", &native.cstring},
					{"CFRelease", &native.release},
				},
			},
		} {
			// Frameworks and the single callback remain loaded for process lifetime.
			handle, err := purego.Dlopen(library.path, purego.RTLD_LAZY|purego.RTLD_LOCAL)
			if err != nil {
				native.err = err
				return
			}
			for _, symbol := range library.symbols {
				address, err := purego.Dlsym(handle, symbol.name)
				if err != nil {
					native.err = err
					return
				}
				purego.RegisterFunc(symbol.target, address)
			}
		}
		native.callback = purego.NewCallback(propertyChanged)
	})
	return native.err
}

// Start subscribes before reading the initial defaults. The returned stop function
// is idempotent, unregisters listeners and waits for active observations to finish.
func Start() (stop func(), err error) {
	if err := loadNative(); err != nil {
		return nil, fmt.Errorf("loading CoreAudio observer: %w", err)
	}
	o := newObserver(readDefault)
	listeners.Lock()
	listeners.next++
	token := listeners.next
	listeners.all[token] = o
	listeners.Unlock()

	registered := 0
	running := false
	var once sync.Once
	stop = func() {
		once.Do(func() {
			o.mu.Lock()
			o.stopped = true
			o.mu.Unlock()
			o.signal()
			listeners.Lock()
			delete(listeners.all, token)
			listeners.Unlock()
			// Never hold a Go mutex across removal: HAL may wait for a callback.
			for index := range registered {
				if status := native.remove(systemObject, &defaultAddresses[index], native.callback, token); status != 0 {
					log.Printf("[audio] unregister_error default=%s source=unknown status=%d", defaultKinds[index], status)
				}
			}
			if running {
				<-o.done
			}
		})
	}
	for index := range defaultAddresses {
		if status := native.add(systemObject, &defaultAddresses[index], native.callback, token); status != 0 {
			stop()
			return nil, fmt.Errorf("subscribing to default %s: CoreAudio status %d", defaultKinds[index], status)
		}
		registered++
	}
	o.initialize()
	running = true
	go o.run()
	return stop, nil
}

func propertyChanged(object uint32, count uint32, addresses *propertyAddress, token uintptr) uintptr {
	if object != systemObject || addresses == nil {
		return 0
	}
	listeners.Lock()
	o := listeners.all[token]
	listeners.Unlock()
	if o == nil {
		return 0
	}
	// Tokens are never reused; late callbacks cannot refer to a newer observer.
	for _, address := range unsafe.Slice(addresses, count) {
		for index, watched := range defaultAddresses {
			if address == watched {
				o.notify(index)
			}
		}
	}
	return 0
}

// readDefault also provides a read-only native baseline for package-local diagnostics.
func readDefault(index int) (device, error) {
	var result device
	size := uint32(unsafe.Sizeof(result.id))
	if status := native.get(systemObject, &defaultAddresses[index], 0, 0, &size, unsafe.Pointer(&result.id)); status != 0 {
		return result, fmt.Errorf("reading device ID: CoreAudio status %d", status)
	}
	if size != uint32(unsafe.Sizeof(result.id)) {
		return result, fmt.Errorf("reading device ID: unexpected size %d", size)
	}
	result.valid = true
	if result.id == 0 {
		result.nameKnown = true // No default device is a valid state.
		return result, nil
	}
	address := propertyAddress{selector: nameSelector, scope: globalScope}
	var name uintptr
	size = uint32(unsafe.Sizeof(name))
	if status := native.get(result.id, &address, 0, 0, &size, unsafe.Pointer(&name)); status != 0 {
		return result, fmt.Errorf("reading device name: CoreAudio status %d", status)
	}
	if name == 0 {
		return result, fmt.Errorf("reading device name: null CFString")
	}
	defer native.release(name)
	buffer := make([]byte, native.length(name)*4+1)
	if !native.cstring(name, buffer, int64(len(buffer)), utf8Encoding) {
		return result, fmt.Errorf("converting device name to UTF-8")
	}
	if end := bytes.IndexByte(buffer, 0); end >= 0 {
		buffer = buffer[:end]
	}
	result.name = string(buffer)
	result.nameKnown = true
	return result, nil
}
