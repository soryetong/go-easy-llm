package utils

import (
	"log"
	"runtime/debug"
)

func GoSafe(fn func()) {
	go RunSafe(fn)
}

func RunSafe(fn func()) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("[go func] panic: %v, stack=%s \n", err, string(debug.Stack()))
		}
	}()

	fn()
}
