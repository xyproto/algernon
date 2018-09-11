package simplebolt

import (
	"github.com/xyproto/pinterface"
)

// BoltCreator is used for implementing pinterface.ICreator. It contains a
// database and provides functions for creating data structures within that
// database.
type BoltCreator struct {
	db *Database
}

// NewCreator can create a new BoltCreator struct
func NewCreator(db *Database) *BoltCreator {
	return &BoltCreator{db}
}

// NewList can create a new List with the given ID
func (b *BoltCreator) NewList(id string) (pinterface.IList, error) {
	return NewList(b.db, id)
}

// NewSet can create a new Set with the given ID
func (b *BoltCreator) NewSet(id string) (pinterface.ISet, error) {
	return NewSet(b.db, id)
}

// NewHashMap can create a new HashMap with the given ID.
// The HashMap elements have a name and then a key+value. For example a
// username for the name, then "password" as the key and a password hash as
// the value.
func (b *BoltCreator) NewHashMap(id string) (pinterface.IHashMap, error) {
	return NewHashMap(b.db, id)
}

// NewKeyValue can create a new KeyValue with the given ID.
// The KeyValue elements have a key with a corresponding value.
func (b *BoltCreator) NewKeyValue(id string) (pinterface.IKeyValue, error) {
	return NewKeyValue(b.db, id)
}
