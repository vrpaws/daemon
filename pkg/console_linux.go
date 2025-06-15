//go:build !windows

package lib

import (
	"log"
)

func DisableQuickEdit() {
	log.Println("Quick edit is not supported on Linux")
}
