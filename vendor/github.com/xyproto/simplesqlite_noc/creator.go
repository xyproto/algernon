package simplesqlite

import (
	"github.com/xyproto/pinterface"
)

// For implementing pinterface.ICreator

type SQLiteCreator struct {
	file *File
}

func NewCreator(file *File) *SQLiteCreator {
	return &SQLiteCreator{file}
}

func (m *SQLiteCreator) NewList(id string) (pinterface.IList, error) {
	return NewList(m.file, id)
}

func (m *SQLiteCreator) NewSet(id string) (pinterface.ISet, error) {
	return NewSet(m.file, id)
}

func (m *SQLiteCreator) NewHashMap(id string) (pinterface.IHashMap, error) {
	return NewHashMap(m.file, id)
}

func (m *SQLiteCreator) NewKeyValue(id string) (pinterface.IKeyValue, error) {
	return NewKeyValue(m.file, id)
}
