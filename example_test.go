package di_test

import "fmt"
import "github.com/AtsushiSuzuki/go-di"

type Logger interface {
	Log(message string)
}

type myLogger struct {
}

func (this *myLogger) Log(message string) {
	fmt.Println(message)
}

type myModule struct {
	Logger Logger `di:"logger"`
}

func (this *myModule) DoWork(name string) {
	this.Logger.Log("Hello, " + name)
}

func init() {
	di.Registry.RegisterType(&myLogger{}, di.Transient)
	di.Registry.RegisterType(&myModule{}, di.Transient)
}

func Example() {
	c := di.Registry.NewScope()
	c.Use("logger", "*di_test.myLogger")
	c.Use("module", "*di_test.myModule")

	v, _ := c.Resolve("module")
	v.(*myModule).DoWork("world")

	// Output:
	// Hello, world
}
