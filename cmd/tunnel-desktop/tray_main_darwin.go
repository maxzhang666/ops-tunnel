//go:build darwin

package main

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework Cocoa
#include <dispatch/dispatch.h>

extern void _cgoTrayMainInit();

static inline void _dispatchToMain() {
	dispatch_async(dispatch_get_main_queue(), ^{
		_cgoTrayMainInit();
	});
}
*/
import "C"

var _pendingMainFn func()

//export _cgoTrayMainInit
func _cgoTrayMainInit() {
	if fn := _pendingMainFn; fn != nil {
		_pendingMainFn = nil
		fn()
	}
}

func runOnMainThread(fn func()) {
	_pendingMainFn = fn
	C._dispatchToMain()
}
