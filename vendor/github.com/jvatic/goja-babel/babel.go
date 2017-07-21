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
	Ready     chan struct{}
}

func (t *babelTransformer) Done() {
	t.Ready <- struct{}{}
}

var mux sync.RWMutex
var pool []*babelTransformer

func Init(poolSize int) error {
	mux.Lock()
	defer mux.Unlock()
	pool = make([]*babelTransformer, poolSize)
	for i := 0; i < poolSize; i++ {
		vm := goja.New()
		transformFn, err := loadBabel(vm)
		if err != nil {
			return err
		}
		pool[i] = &babelTransformer{Runtime: vm, Transform: transformFn, Ready: make(chan struct{}, 1)}
		pool[i].Ready <- struct{}{} // Transformer available for use
	}
	return nil
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
	if len(pool) == 0 {
		if err := Init(DefaultPoolSize); err != nil {
			return nil, err
		}
	}
	mux.RLock()
	defer mux.RUnlock()
	for {
		// find first available transformer
		for _, t := range pool {
			select {
			case <-t.Ready:
				return t, nil
			default:
			}
		}
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
