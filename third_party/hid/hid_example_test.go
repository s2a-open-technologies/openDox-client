package hid_test

import (
	"fmt"

	"github.com/ricknicius99/hid"
)

func ExampleSupported() {
	fmt.Println("Build configuration supported:", hid.Supported())
	// Output:
	// Build configuration supported: true
}
