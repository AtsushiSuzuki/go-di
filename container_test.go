package di

import "testing"
import "github.com/stretchr/testify/assert"

type MyStruct struct {
	Name  string
	Value string
}

func TestRegisterType(t *testing.T) {
	c := newContainer(nil)

	var a *MyStruct
	c.RegisterType(a, Transient)

	assert.Equal(t, 1, len(c.factories))
	f := c.factories[0]
	assert.Equal(t, "*di.MyStruct", f.Tag)
	assert.Equal(t, Transient, f.Lifetime)
	assert.NotNil(t, f.Constructor)
	assert.Nil(t, f.Destructor)
}

func TestResolve_RegisteredType_Struct(t *testing.T) {
	c := newContainer(nil)

	var a MyStruct
	c.RegisterType(a, Transient)

	v, err := c.Resolve("di.MyStruct")
	assert.NoError(t, err)
	assert.IsType(t, MyStruct{}, v)
	assert.Equal(t, MyStruct{}, v)
}

func TestResolve_RegisteredType_PointerToStruct(t *testing.T) {
	c := newContainer(nil)

	var a *MyStruct
	c.RegisterType(a, Transient)

	v, err := c.Resolve("*di.MyStruct")
	assert.NoError(t, err)
	assert.IsType(t, &MyStruct{}, v)
	assert.Equal(t, MyStruct{}, *v.(*MyStruct))

	v2, err := c.Resolve("*di.MyStruct")
	assert.NoError(t, err)
	assert.True(t, v != v2)
}

func TestResolve_RegisteredType_Primitive(t *testing.T) {
	c := newContainer(nil)

	var a int
	c.RegisterType(a, Transient)

	v, err := c.Resolve("int")
	assert.NoError(t, err)
	assert.IsType(t, 0, v)
	assert.Equal(t, 0, v)
}

func TestResolve_RegisteredType_Slice(t *testing.T) {
	c := newContainer(nil)

	var a []MyStruct
	c.RegisterType(a, Transient)

	v, err := c.Resolve("[]di.MyStruct")
	assert.NoError(t, err)
	assert.IsType(t, make([]MyStruct, 0), v)
	// assert.Equal(t, make([]MyStruct, 0), v)
	assert.Equal(t, []MyStruct(nil), v)
}

func TestRegisterValue(t *testing.T) {
	c := newContainer(nil)

	c.RegisterValue(&MyStruct{})

	assert.Equal(t, 1, len(c.factories))
	f := c.factories[0]
	assert.Equal(t, "*di.MyStruct", f.Tag)
	assert.Equal(t, Transient, f.Lifetime)
	assert.NotNil(t, f.Constructor)
	assert.Nil(t, f.Destructor)
}

func TestResolve_RegisteredValue_PointerToStruct(t *testing.T) {
	c := newContainer(nil)

	a := &MyStruct{}
	c.RegisterValue(a)

	v, err := c.Resolve("*di.MyStruct")
	assert.NoError(t, err)
	assert.IsType(t, &MyStruct{}, v)
	assert.True(t, a == v)
}

func TestResolve_RegisteredValue_String(t *testing.T) {
	c := newContainer(nil)

	c.UseValue("greet", "Hello, world!")

	v, err := c.Resolve("greet")
	assert.NoError(t, err)
	assert.Equal(t, "Hello, world!", v)
}

func TestRegisterFactory(t *testing.T) {
	c := newContainer(nil)

	var a *MyStruct
	ctor := func(c Container) (interface{}, error) {
		return &MyStruct{}, nil
	}
	dtor := func(v interface{}) error {
		return nil
	}
	c.RegisterFactory(a, ctor, dtor, Transient)

	assert.Equal(t, 1, len(c.factories))
	f := c.factories[0]
	assert.Equal(t, "*di.MyStruct", f.Tag)
	assert.Equal(t, Transient, f.Lifetime)
	assert.Equal(t, Constructor(ctor), f.Constructor)
	assert.Equal(t, Destructor(dtor), f.Destructor)
}

func TestResolve_RegisteredFactory(t *testing.T) {
	c := newContainer(nil)

	var a *MyStruct
	c.RegisterFactory(a, func(c Container) (interface{}, error) {
		return &MyStruct{}, nil
	}, nil, Transient)

	v, err := c.Resolve("*di.MyStruct")
	assert.NoError(t, err)
	assert.Equal(t, &MyStruct{}, v)
	assert.Equal(t, 0, len(c.destructors))
}

func TestResolve_RegisteredFactory_WithDestructor(t *testing.T) {
	c := newContainer(nil)

	var a *MyStruct
	c.RegisterFactory(a, func(c Container) (interface{}, error) {
		return &MyStruct{}, nil
	}, func(v interface{}) error {
		return nil
	}, Transient)

	v, err := c.Resolve("*di.MyStruct")
	assert.NoError(t, err)
	assert.Equal(t, &MyStruct{}, v)
	assert.Equal(t, 1, len(c.destructors))
}

func TestClose(t *testing.T) {
	c := newContainer(nil)

	var a *MyStruct
	closed := false
	c.RegisterFactory(a, func(c Container) (interface{}, error) {
		return &MyStruct{}, nil
	}, func(v interface{}) error {
		closed = true
		return nil
	}, Transient)

	v, err := c.Resolve("*di.MyStruct")
	assert.NoError(t, err)
	assert.Equal(t, &MyStruct{}, v)
	assert.Equal(t, false, closed)

	c.Close()
	assert.Equal(t, true, closed)
}

func TestResolve_Hierarchical(t *testing.T) {
	root := newContainer(nil)
	c := newContainer(root)

	root.UseValue("greet", "hello")

	v, err := c.Resolve("greet")
	assert.NoError(t, err)
	assert.Equal(t, "hello", v)
}

func TestUse(t *testing.T) {
	c := newContainer(nil)

	c.Use("struct", "*di.MyStruct")

	assert.Equal(t, 1, len(c.aliases))
	alias := c.aliases[0]
	assert.Equal(t, "struct", alias.Tag)
	assert.Equal(t, "*di.MyStruct", alias.Aliased)
}

func TestResolve_Aliased(t *testing.T) {
	c := newContainer(nil)

	c.RegisterType(&MyStruct{}, Transient)
	c.Use("struct", "*di.MyStruct")

	v, err := c.Resolve("struct")
	assert.NoError(t, err)
	assert.Equal(t, &MyStruct{}, v)
}

func TestResolve_Aliased_Double(t *testing.T) {
	c := newContainer(nil)

	c.RegisterType(&MyStruct{}, Transient)
	c.Use("struct", "*di.MyStruct")
	c.Use("param", "struct")

	v, err := c.Resolve("param")
	assert.NoError(t, err)
	assert.Equal(t, &MyStruct{}, v)
}

func TestResolve_Aliased_Hierarchical(t *testing.T) {
	root := newContainer(nil)
	c := newContainer(root)

	root.RegisterType(&MyStruct{}, Transient)
	root.Use("struct", "*di.MyStruct")
	c.Use("param", "struct")

	v, err := c.Resolve("param")
	assert.NoError(t, err)
	assert.Equal(t, &MyStruct{}, v)
}

type MyStruct2 struct {
	*MyStruct `di:"*di.MyStruct"`
}

func TestResolve_InjectField(t *testing.T) {
	c := newContainer(nil)

	var a *MyStruct2
	c.RegisterType(a, Transient)
	var a2 *MyStruct
	c.RegisterType(a2, Transient)

	v, err := c.Resolve("*di.MyStruct2")
	assert.NoError(t, err)
	assert.NotNil(t, v.(*MyStruct2).MyStruct)
	assert.Equal(t, &MyStruct2{&MyStruct{}}, v)
}

func TestResolve_MultipleRegistration(t *testing.T) {
	c := newContainer(nil)

	c.UseValue("params", 1)
	c.UseValue("params", 2)
	c.UseValue("params", 3)

	v, err := c.Resolve("params")
	assert.NoError(t, err)
	assert.Equal(t, 3, v)
}

func TestResolve_ScopedCache(t *testing.T) {
	root := newContainer(nil)
	c := newContainer(root)

	root.RegisterType(&MyStruct{}, Scoped)

	v, err := c.Resolve("*di.MyStruct")
	assert.NoError(t, err)
	assert.Equal(t, &MyStruct{}, v)

	v2, err := c.Resolve("*di.MyStruct")
	assert.NoError(t, err)
	assert.True(t, v == v2)

	c2 := newContainer(root)
	v3, err := c2.Resolve("*di.MyStruct")
	assert.NoError(t, err)
	assert.True(t, v != v3)
}

func TestResolve_SingletonCache(t *testing.T) {
	root := newContainer(nil)
	c := newContainer(root)

	root.RegisterType(&MyStruct{}, Singleton)

	v, err := c.Resolve("*di.MyStruct")
	assert.NoError(t, err)
	assert.Equal(t, &MyStruct{}, v)

	v2, err := c.Resolve("*di.MyStruct")
	assert.NoError(t, err)
	assert.True(t, v == v2)

	c2 := newContainer(root)
	v3, err := c2.Resolve("*di.MyStruct")
	assert.NoError(t, err)
	assert.True(t, v == v3)
}

func TestResolve_NotFound(t *testing.T) {
	c := newContainer(nil)

	_, err := c.Resolve("*di.MyStruct")
	assert.Error(t, err)
}

func TestResolveAll(t *testing.T) {
	c := newContainer(nil)

	c.UseValue("params", 1)
	c.UseValue("params", 2)
	c.UseValue("params", 3)

	v, err := c.ResolveAll("params")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(v))
	assert.Equal(t, 1, v[0])
	assert.Equal(t, 2, v[1])
	assert.Equal(t, 3, v[2])
}
