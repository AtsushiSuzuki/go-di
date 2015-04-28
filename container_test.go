package di

import "testing"
import "github.com/stretchr/testify/assert"

type MyStruct struct {
	A string
	B []string
}

func TestResolveType(t *testing.T) {
	Registry.RegisterType(&MyStruct{}, Transient)

	c := Registry.NewScope()
	c.Use("struct", "*di.MyStruct")
	v, err := c.Resolve("struct")
	assert.NoError(t, err)
	t.Logf("%v (%T)", v, v)
}

type MyStruct2 struct {
	Name string `di:"param"`
}

func TestResolveValue(t *testing.T) {
	c := Registry.NewScope()
	c.UseType("struct", &MyStruct2{}, Transient)
	c.UseValue("param", "hello, world!", Transient)

	v, err := c.Resolve("struct")
	assert.NoError(t, err)
	assert.Equal(t, "hello, world!", v.(*MyStruct2).Name)
}
