package gpgme

// #include <stdlib.h>
import "C"

import (
	"sync"
	"unsafe"
)

/* This file implements a glue between callback in Go (*func, a closure), and C (function pointer + void *).

The primary issue this needs to solve is the following restriction in https://golang.org/cmd/cgo/ :

> Go code may pass a Go pointer to C...
> _C code may not keep a copy of a Go pointer after the call returns._

So we cannot simply register a C callback with the void * parameter pointing to a Go object (neither a closure nor a struct),
because when the "register callback" function returns, the C code must not keep a copy of that pointer.

So, this Go code maintains a hash table indexed by $something_we_can_pass_to_C and storing the "Go pointers"; we pass
$something... to C, and when C calls our callback with $something, our Go code resolves it back to a "Go pointer".


As for $something_we_can_pass_to_C, the cgo document linked above only says that it should not be a "Go pointer";
i.e. an int or uintptr value would be fine; and in fact (void *)(uintptr_t)some_integer_value would be a natural
way to do this in C.  But there is another restriction in https://golang.org/pkg/unsafe/ :

> The remaining patterns enumerate the only valid conversions from uintptr to Pointer.
> Conversion of a Pointer to a uintptr and back, with arithmetic.
... and other irrelevant cases.

In other words, we can't index the hash table by an int or uintptr, because e.g. unsafe.Pointer(1) is prohibited
(although the compiler, as of Go 1.5 and 1.6, does not warn about it).  And because, per the discussion above,
$something_we_can_pass_to_C must not be a "Go pointer", the only remaining option is to use C pointers as the
hash table indexes.

So, we call C.malloc(1) to allocate a unique C pointer = hash key. The allocation/deallocation is a bit costly,
OTOH at least we don't need to worry about int wraparound and duplicated index values. */

var callbacks struct {
	sync.Mutex
	m map[unsafe.Pointer]interface{}
}

func callbackAdd(v interface{}) unsafe.Pointer {
	ret := C.malloc(1)
	if ret == nil {
		panic("malloc failed")
	}

	callbacks.Lock()
	defer callbacks.Unlock()
	if callbacks.m == nil {
		callbacks.m = make(map[unsafe.Pointer]interface{})
	}
	callbacks.m[ret] = v
	return ret
}

func callbackLookup(c unsafe.Pointer) interface{} {
	callbacks.Lock()
	defer callbacks.Unlock()
	ret := callbacks.m[c]
	if ret == nil {
		panic("callback pointer not found")
	}
	return ret
}

func callbackDelete(c unsafe.Pointer) {
	callbacks.Lock()
	defer callbacks.Unlock()
	if callbacks.m[c] == nil {
		panic("callback pointer not found")
	}
	delete(callbacks.m, c)

	C.free(c)
}
