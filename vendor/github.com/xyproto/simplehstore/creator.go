package simplehstore

import (
	"github.com/xyproto/pinterface"
)

// PostgresCreator is a general struct to create datatypes with.
// The main purpose is to implement pinterface.ICreator.
type PostgresCreator struct {
	host *Host
}

// NewCreator can be used to create a new PostgresCreator.
// The main purpose is to implement pinterface.ICreator.
func NewCreator(host *Host) *PostgresCreator {
	return &PostgresCreator{host}
}

// NewList can be used to create a new pinterface.IList.
// The main purpose is to implement pinterface.ICreator.
func (m *PostgresCreator) NewList(id string) (pinterface.IList, error) {
	return NewList(m.host, id)
}

// NewSet can be used to create a new pinterface.ISet.
// The main purpose is to implement pinterface.ICreator.
func (m *PostgresCreator) NewSet(id string) (pinterface.ISet, error) {
	return NewSet(m.host, id)
}

// NewHashMap can be used to create a new pinterface.IHashMap.
// The main purpose is to implement pinterface.ICreator.
func (m *PostgresCreator) NewHashMap(id string) (pinterface.IHashMap, error) {
	return NewHashMap(m.host, id)
}

// NewKeyValue can be used to create a new pinterface.IKeyValue.
// The main purpose is to implement pinterface.ICreator.
func (m *PostgresCreator) NewKeyValue(id string) (pinterface.IKeyValue, error) {
	return NewKeyValue(m.host, id)
}
