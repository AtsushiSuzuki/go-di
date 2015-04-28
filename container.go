package di

import "fmt"
import "reflect"
import "sync"

// Registry is global registry of types being resolved by container.
// Because golang lacks global type registry, you should manually
// register types by `RegisterType`, `RegisterValue` or `RegisterFactory`
// methods.
var Registry Container = &container{
	nil,
	make([]*alias, 0),
	make([]*factory, 0),
	sync.RWMutex{},
}

type Container interface {
	NewScope() Container
	RegisterType(v interface{}, lifetime Lifetime)
	RegisterValue(v interface{}, lifetime Lifetime)
	RegisterFactory(v interface{}, ctor Constructor, dtor Destructor, lifetime Lifetime)
	Use(tag string, tagOrTypeName string)
	UseType(tag string, v interface{}, lifetime Lifetime)
	UseValue(tag string, v interface{}, lifetime Lifetime)
	UseFactory(tag string, ctor Constructor, dtor Destructor, lifetime Lifetime)
	Resolve(tag string) (interface{}, error)
	ResolveAll(tag string) ([]interface{}, error)
	Inject(v interface{}) error
	Close() error
}

type Lifetime int

const (
	Transient Lifetime = iota
	Scoped
	Singleton
)

type Constructor func(c Container) (interface{}, error)

type Destructor func(i interface{}) error

type container struct {
	parent    *container
	aliases   []*alias
	factories []*factory
	lock      sync.RWMutex
}

type alias struct {
	Tag     string
	Aliased string
}

type factory struct {
	Tag         string
	Lifetime    Lifetime
	Constructor Constructor
	Destructor  Destructor
}

func (this *container) NewScope() Container {
	return &container{
		this,
		make([]*alias, 0),
		make([]*factory, 0),
		sync.RWMutex{},
	}
}

func (this *container) RegisterType(v interface{}, lifetime Lifetime) {
	t := reflect.TypeOf(v)
	this.UseType(t.String(), v, lifetime)
}

func (this *container) RegisterValue(v interface{}, lifetime Lifetime) {
	t := reflect.TypeOf(v)
	this.UseValue(t.String(), v, lifetime)
}

func (this *container) RegisterFactory(v interface{}, ctor Constructor, dtor Destructor, lifetime Lifetime) {
	t := reflect.TypeOf(v)
	this.UseFactory(t.String(), ctor, dtor, lifetime)
}

func (this *container) UseType(tag string, v interface{}, lifetime Lifetime) {
	this.lock.Lock()
	defer this.lock.Unlock()

	typ := reflect.TypeOf(v)
	this.factories = append(this.factories, &factory{
		Tag:      tag,
		Lifetime: lifetime,
		Constructor: func(c Container) (interface{}, error) {
			var instanceType reflect.Type
			if typ.Kind() == reflect.Ptr {
				instanceType = typ.Elem()
			} else {
				instanceType = typ
			}

			ptrInstance := reflect.New(instanceType)
			if instanceType.Kind() == reflect.Struct {
				c.Inject(ptrInstance.Interface())
			}

			if typ.Kind() == reflect.Ptr {
				return ptrInstance.Interface(), nil
			} else {
				return reflect.Indirect(ptrInstance).Interface(), nil
			}
		},
		Destructor: nil,
	})
}

func (this *container) UseValue(tag string, v interface{}, lifetime Lifetime) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.factories = append(this.factories, &factory{
		Tag:      tag,
		Lifetime: lifetime,
		Constructor: func(c Container) (interface{}, error) {
			return v, nil
		},
		Destructor: nil,
	})
}

func (this *container) UseFactory(tag string, ctor Constructor, dtor Destructor, lifetime Lifetime) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.factories = append(this.factories, &factory{
		Tag:         tag,
		Lifetime:    lifetime,
		Constructor: ctor,
		Destructor:  dtor,
	})
}

func (this *container) Use(tag string, tagOrTypeName string) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.aliases = append(this.aliases, &alias{
		Tag:     tag,
		Aliased: tagOrTypeName,
	})
}

type tags []string

func (this tags) Contains(tag string) bool {
	for _, x := range this {
		if x == tag {
			return true
		}
	}
	return false
}

func (this *container) resolveAliases(tag string) tags {
	tags := []string{tag}

	for i := 0; i < len(tags); i++ {
		for c := this; c != nil; c = c.parent {
			for _, alias := range c.aliases {
				if tags[i] != alias.Tag {
					continue
				}

				found := false
				for j := 0; j < len(tags); j++ {
					if alias.Aliased == tags[j] {
						found = true
						break
					}
				}
				if found {
					continue
				}

				tags = append(tags, alias.Aliased)
			}
		}
	}

	return tags
}

func (this *container) Resolve(tag string) (interface{}, error) {
	factory, found := func() (*factory, bool) {
		this.lock.RLock()
		defer this.lock.RUnlock()

		tags := this.resolveAliases(tag)

		for c := this; c != nil; c = c.parent {
			for i := len(c.factories) - 1; 0 <= i; i-- {
				if tags.Contains(c.factories[i].Tag) {
					return c.factories[i], true
				}
			}
		}

		return nil, false
	}()

	if found {
		return factory.Constructor(this)
	} else {
		return nil, fmt.Errorf("no matching tag found.")
	}
}

func (this *container) ResolveAll(tag string) ([]interface{}, error) {
	factories := make([]*factory, 0, 10)
	func() {
		this.lock.RLock()
		defer this.lock.RUnlock()

		tags := this.resolveAliases(tag)

		for c := this; c != nil; c = c.parent {
			for i := len(c.factories) - 1; 0 <= i; i-- {
				if tags.Contains(c.factories[i].Tag) {
					factories = append(factories, c.factories[i])
				}
			}
		}
	}()

	var instances []interface{}
	var err error
	for _, factory := range factories {
		i, err := factory.Constructor(this)
		if err != nil {
			return nil, err
		}
		instances = append(instances, i)
	}

	return instances, nil
}

func (this *container) Inject(v interface{}) error {
	v := reflect.ValueOf(i)
	if v.Type().Kind() == reflect.Ptr {
		v = reflect.Indirect(v)
	}
	if v.Type().Kind() != reflect.Struct {
		panic("struct or pointer to struct required")
	}
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if tag := f.Tag.Get("di"); tag != "" {
			fi, err := this.Resolve(tag)
			if err != nil {
				return err
			}
			v.Field(i).Set(reflect.ValueOf(fi))
		}
	}

	return nil
}

func (this *container) Close() error {
	panic("not implemented")
}
