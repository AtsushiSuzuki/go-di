package di_test

import "fmt"
import "github.com/AtsushiSuzuki/go-di"

type MyStruct struct {
}

func Example_simple() {
	c := di.Registry.NewScope()
	c.RegisterType(&MyStruct{}, di.Transient)

	v, _ := c.Resolve("*di_test.MyStruct")
	fmt.Printf("v: %v (%T)", v, v)

	// Output:
	// v: &{} (*di_test.MyStruct)
}
