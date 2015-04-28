// Package `di` implements basic dependency injection (DI) container.
package di

import "fmt"
import "reflect"
import "sync"

// `Lifetime` represents lifetime of instance resolved by container.
type Lifetime int

const (
	// `Transient` lifetime specifies that instances should
	// be created each time container resolves dependency.
	Transient Lifetime = iota

	// `Scoped` lifetime specifies that the instance should be reused
	// by the same container.
	Scoped

	// `Singleton` lifetime specifies that the instance should be
	// reused for process lifetime.
	Singleton
)

// `Constructor` is generic function type to create instance.
//
// `Container` does not support constructor injection.
// Instead, you can use supplied `Container` to
// wire up your instance's dependency.
type Constructor func(c Container) (interface{}, error)

// `Destructor` is cleanup function for created instance.
type Destructor func(i interface{}) error

// `Container` is type registry and dependency resolver.
//
// You can use `RegisterType`, `RegisterValue` or `RegisterFactory` to
// register types to the container, and then use `Resolve` to obtain
// an registered instance.
//
// `Container` instance is obtained by `Registry.NewScope()`.
type Container interface {
	// `NewScope` creates new scoped container from parent container.
	//
	// Created scope inherits type registrations from it's parent.
	NewScope() Container

	// `RegisterType` registers type (`reflect.TypeOf(v)`) for
	// it's type name (`reflect.TypeOf(v).String`).
	//
	// When resolved, new zero/nil instance is created.
	// If type is pointer, type is resolved as an pointer points to
	// newly created element instead of nil pointer.
	//
	// If created instance is struct, it's field with
	// tag `di:"<tag>"` will be injected. see `Inject`.
	RegisterType(v interface{}, lifetime Lifetime)

	// `RegisterValue` registers supplied value `v` for
	// it's type name.
	RegisterValue(v interface{})

	// `RegisterFactory` registers factory method for type name of `v`.
	//
	// You can optionally use destructor. If supplied, `dtor` is called
	// for each resolved instance when asociated container is `Close`ed.
	//
	// TODO: inspect type name from constructor return value
	RegisterFactory(v interface{}, ctor Constructor, dtor Destructor, lifetime Lifetime)

	// `Use` sets an alias for other tag or type name.
	Use(tag string, tagOrTypeName string)

	// `UseType` registers type for specified tag.
	//
	// See `RegisterType`.
	UseType(tag string, v interface{}, lifetime Lifetime)

	// `UseValue` registers value for specified tag.
	//
	// See `RegisterValue`
	UseValue(tag string, v interface{})

	// `UseFactory` registers factory method for specified tag.
	//
	// See `RegisterFactory`.
	UseFactory(tag string, ctor Constructor, dtor Destructor, lifetime Lifetime)

	// `Resolve` returns instance for specified tag.
	//
	// If multiple type is registered for tag,
	// `Resolve` returns instance of last registered type.
	//
	// If no type is registered for tag, returns `ErrNoMatchingTag`.
	Resolve(tag string) (interface{}, error)

	// `ResolveAll` returns slice of instances for specified tag.
	ResolveAll(tag string) ([]interface{}, error)

	// `Inject` fills struct `v`'s fields with resolved instances
	// if field with tagged as `di:"<tag>"`
	Inject(v interface{}) error

	// `Close` invokes destructors for all instances resolved by
	// the container.
	Close() error
}

// `Registry` is global registry of types being resolved by containers.
var Registry Container = newContainer(nil)

var ErrNoMatchingTag error = fmt.Errorf("no matching tag found")

type container struct {
	parent      *container
	aliases     []*alias
	factories   []*factory
	cache       map[*factory]interface{}
	destructors []func() error
	lock        sync.RWMutex
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

func newContainer(parent *container) *container {
	return &container{
		parent,
		make([]*alias, 0),
		make([]*factory, 0),
		make(map[*factory]interface{}),
		make([]func() error, 0),
		sync.RWMutex{},
	}
}

func (this *container) NewScope() Container {
	return newContainer(this)
}

func (this *container) RegisterType(v interface{}, lifetime Lifetime) {
	t := reflect.TypeOf(v)
	this.UseType(t.String(), v, lifetime)
}

func (this *container) RegisterValue(v interface{}) {
	t := reflect.TypeOf(v)
	this.UseValue(t.String(), v)
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

func (this *container) UseValue(tag string, v interface{}) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.factories = append(this.factories, &factory{
		Tag:      tag,
		Lifetime: Transient,
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

func (this *container) createInstance(f *factory) (interface{}, error) {
	root := func(c *container) *container {
		for ; c.parent != nil; c = c.parent {
		}
		return c
	}(this)

	switch f.Lifetime {
	case Scoped:
		this.lock.RLock()
		cached, ok := this.cache[f]
		this.lock.RUnlock()
		if ok {
			return cached, nil
		}
	case Singleton:
		root.lock.RLock()
		cached, ok := root.cache[f]
		root.lock.RUnlock()
		if ok {
			return cached, nil
		}
	}

	instance, err := f.Constructor(this)
	if err != nil {
		return instance, err
	}

	switch f.Lifetime {
	case Scoped:
		this.lock.Lock()
		this.cache[f] = instance
		this.lock.Unlock()
	case Singleton:
		root.lock.Lock()
		root.cache[f] = instance
		root.lock.Unlock()
	}

	if f.Destructor != nil {
		switch f.Lifetime {
		case Transient:
			fallthrough
		case Scoped:
			this.lock.Lock()
			this.destructors = append(this.destructors, func() error {
				return f.Destructor(instance)
			})
			this.lock.Unlock()
		case Singleton:
			root.lock.Lock()
			root.destructors = append(root.destructors, func() error {
				return f.Destructor(instance)
			})
			root.lock.Unlock()
		}
	}

	return instance, nil
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
		return this.createInstance(factory)
	} else {
		return nil, ErrNoMatchingTag
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
	for i := len(factories) - 1; 0 <= i; i-- {
		v, err := this.createInstance(factories[i])
		if err != nil {
			return nil, err
		}
		instances = append(instances, v)
	}

	return instances, nil
}

func (this *container) Inject(v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Type().Kind() == reflect.Ptr {
		val = reflect.Indirect(val)
	}
	if val.Type().Kind() != reflect.Struct {
		panic("struct or pointer to struct required")
	}
	t := val.Type()

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if tag := f.Tag.Get("di"); tag != "" {
			fi, err := this.Resolve(tag)
			if err != nil {
				return err
			}
			val.Field(i).Set(reflect.ValueOf(fi))
		}
	}

	return nil
}

func (this *container) Close() error {
	this.lock.Lock()
	dtors := make([]func() error, len(this.destructors))
	copy(dtors, this.destructors)
	this.destructors = this.destructors[:0]
	this.lock.Unlock()

	for i := len(dtors) - 1; 0 <= i; i-- {
		err := dtors[i]()
		if err != nil {
			return err
		}
	}

	return nil
}
