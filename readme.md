# go-di

Dependency Injection container library for golang.

https://github.com/AtsushiSuzuki/go-di

http://gidoc.org/github.com/AtsushiSuzuki/go-di

## Features

- Register type/value to container, retrieve registered instance by type name.
- Set tag (alias) to type/value.
- Inject struct field by struct tag.
- Hierarchical container.
- Manage instance lifetime by container. (Transient, Scoped, Singleton)

## Example

```go
package main

import "fmt"
import "github.com/AtsushiSuzuki/go-di"

// define inteface "Logger" and struct "myLogger"
type Logger interface {
    Log(message string)
}

type myLogger struct {
}

func (this *myLogger) Log(message string) {
    fmt.Println(message)
}

// define struct "myModule"
type myModule struct {
    Logger Logger `di:"logger"` // declare dependency to `logger`
}

func (this *myModule) DoWork(name string) {
    this.Logger.Log("Hello, " + name)
}

// register types
func init() {
    di.Registry.RegisterType(&myLogger{}, di.Transient)
    di.Registry.RegisterType(&myModule{}, di.Transient)
}

func main() {
    c := di.Registry.NewScope()
    c.Use("logger", "*main.myLogger")
    c.Use("module", "*main.myModule")

    v, _ := c.Resolve("module")
    v.(*myModule).DoWork("world")

    // Output:
    // Hello, world
}

```

## API document
see [godoc (http://godoc.org/github.com/AtsushiSuzuki/go-di)](http://godoc.org/github.com/AtsushiSuzuki/go-di).

## Limitation
- Requires type registration by hand.
- No constructor injection.
- No property injection.

## TODOs
- Better Container.RegisterFactory API.
- Multithreading test.
- Refactor lockings.
- Check for other DI libraries.

## License

[ISC (http://opensource.org/licenses/ISC)](http://opensource.org/licenses/ISC)