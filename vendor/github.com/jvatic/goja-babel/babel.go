package babel

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/dop251/goja"
)

const DefaultPoolSize = 1

type babelTransformer struct {
	Runtime   *goja.Runtime
	Transform func(string, map[string]interface{}) (goja.Value, error)
}

func (t *babelTransformer) Done() {
	globalpool <- t
}

var once = &sync.Once{}
var globalpool chan *babelTransformer

func Init(poolSize int) (err error) {
	once.Do(func() {
		globalpool = make(chan *babelTransformer, poolSize)
		for i := 0; i < poolSize; i++ {
			vm := goja.New()
			transformFn, e := loadBabel(vm)
			if e != nil {
				err = e
				return
			}
			globalpool <- &babelTransformer{Runtime: vm, Transform: transformFn}
		}
	})

	return err
}

func Transform(src io.Reader, opts map[string]interface{}) (io.Reader, error) {
	data, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}
	res, err := TransformString(string(data), opts)
	if err != nil {
		return nil, err
	}
	return strings.NewReader(res), nil
}

func TransformString(src string, opts map[string]interface{}) (string, error) {
	if opts == nil {
		opts = map[string]interface{}{}
	}
	t, err := getTransformer()
	if err != nil {
		return "", err
	}
	defer func() { t.Done() }() // Make transformer available again when we're done
	v, err := t.Transform(src, opts)
	if err != nil {
		return "", err
	}
	vm := t.Runtime
	return v.ToObject(vm).Get("code").String(), nil
}

func getTransformer() (*babelTransformer, error) {
	// Make sure we have a pool created
	if len(globalpool) == 0 {
		if err := Init(DefaultPoolSize); err != nil {
			return nil, err
		}
	}
	for {
		t := <-globalpool
		return t, nil
	}
}

func loadBabel(vm *goja.Runtime) (func(string, map[string]interface{}) (goja.Value, error), error) {
	babelsrc, err := _Asset("babel.js")
	if err != nil {
		return nil, err
	}
	_, err = vm.RunScript("babel.js", string(babelsrc))
	if err != nil {
		return nil, fmt.Errorf("unable to load babel.js: %s", err)
	}
	var transform goja.Callable
	babel := vm.Get("Babel")
	if err := vm.ExportTo(babel.ToObject(vm).Get("transform"), &transform); err != nil {
		return nil, fmt.Errorf("unable to export transform fn: %s", err)
	}
	return func(src string, opts map[string]interface{}) (goja.Value, error) {
		return transform(babel, vm.ToValue(src), vm.ToValue(opts))
	}, nil
}
