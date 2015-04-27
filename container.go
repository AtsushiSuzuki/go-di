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
	make([]*factory, 0),
	sync.RWMutex{},
}

type Container interface {
	NewScope() Container
	RegisterType(v interface{})
	RegisterValue(v interface{})
	RegisterFactory(v interface{}, ctor Constructor, dtor Destructor)
	Use(tag string, typeName string)
	UseType(tag string, v interface{})
	UseValue(tag string, v interface{})
	UseFactory(tag string, ctor Constructor, dtor Destructor)
	Resolve(tag string) (interface{}, error)
	ResolveAll(tag string) ([]interface{}, error)
	Inject(v interface{}) error
}

type Constructor func(c Container) (interface{}, error)

type Destructor func(i interface{}) error

type container struct {
	parent    *container
	factories []*factory
	lock      sync.RWMutex
}

type factory struct {
	Tag         string
	Constructor Constructor
	Destructor  Destructor
}

func (this *container) NewScope() Container {
	return &container{
		this,
		make([]*factory, 0),
		sync.RWMutex{},
	}
}

func (this *container) RegisterType(v interface{}) {
	t := reflect.TypeOf(v)
	this.UseType(t.String(), v)
}

func (this *container) RegisterValue(v interface{}) {
	t := reflect.TypeOf(v)
	this.UseValue(t.String(), v)
}

func (this *container) RegisterFactory(v interface{}, ctor Constructor, dtor Destructor) {
	t := reflect.TypeOf(v)
	this.UseFactory(t.String(), ctor, dtor)
}

func (this *container) UseType(tag string, v interface{}) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.factories = append(this.factories, &factory{
		Tag: tag,
		Constructor: func(c Container) (interface{}, error) {
			typ := reflect.TypeOf(v)

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
		Tag: tag,
		Constructor: func(c Container) (interface{}, error) {
			return v, nil
		},
		Destructor: nil,
	})
}

func (this *container) UseFactory(tag string, ctor Constructor, dtor Destructor) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.factories = append(this.factories, &factory{
		Tag:         tag,
		Constructor: ctor,
		Destructor:  dtor,
	})
}

func (this *container) Use(tag string, typeName string) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.factories = append(this.factories, &factory{
		Tag: tag,
		Constructor: func(c Container) (interface{}, error) {
			return this.Resolve(typeName)
		},
		Destructor: nil,
	})
}

func (this *container) Resolve(tag string) (interface{}, error) {
	factory, found := func() (*factory, bool) {
		this.lock.RLock()
		defer this.lock.RUnlock()

		for i := len(this.factories) - 1; 0 <= i; i-- {
			if tag == this.factories[i].Tag {
				return this.factories[i], true
			}
		}
		return nil, false
	}()

	if found {
		return factory.Constructor(this)
	} else if this.parent != nil {
		return this.parent.Resolve(tag)
	} else {
		return nil, fmt.Errorf("no matching tag found.")
	}
}

func (this *container) ResolveAll(tag string) ([]interface{}, error) {
	factories := make([]*factory, 0, 10)
	func() {
		this.lock.RLock()
		defer this.lock.RUnlock()

		for i := 0; i < len(this.factories); i++ {
			if tag == this.factories[i].Tag {
				factories = append(factories, this.factories[i])
			}
		}
	}()

	var instances []interface{}
	var err error
	if this.parent != nil {
		instances, err = this.parent.ResolveAll(tag)
		if err != nil {
			return nil, err
		}
	} else {
		instances = make([]interface{}, 0, 10)
	}

	for _, factory := range factories {
		i, err := factory.Constructor(this)
		if err != nil {
			return nil, err
		}
		instances = append(instances, i)
	}

	return instances, nil
}

func (this *container) Inject(i interface{}) error {
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
