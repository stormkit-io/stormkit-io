package factory

import (
	"reflect"

	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
)

type MockObject interface {
	Insert(conn databasetest.TestDB) error
}

type Factory struct {
	conn    databasetest.TestDB
	objects []MockObject // A reference to all objects created
}

func New(conn databasetest.TestDB) *Factory {
	return &Factory{
		conn:    conn,
		objects: []MockObject{},
	}
}

func (f *Factory) newObject(m MockObject) MockObject {
	f.objects = append(f.objects, m)
	return m
}

func factoryLookup[T any](f *Factory) *T {
	if f == nil || f.objects == nil {
		return nil
	}

	for _, object := range f.objects {
		if _, ok := object.(T); ok {
			mock := object.(T)
			return &mock
		}
	}

	return nil
}

func merge(obj interface{}, values map[string]any) {
	st := reflect.ValueOf(obj).Elem()

	for k, v := range values {
		f := st.FieldByName(k)
		v := reflect.ValueOf(v)
		f.Set(v)
	}
}
