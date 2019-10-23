// skip : needs some synchronization mechanism. https://github.com/open2b/scriggo/issues/420

// run

package main

import (
	"fmt"
	"time"
)

func main() {
	go func() {
		fmt.Print("func literal")
	}()
	time.Sleep(1 * time.Millisecond)
}
