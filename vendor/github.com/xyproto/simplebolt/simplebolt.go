// Package simplebolt provides a simple way to use the Bolt database.
// The API design is similar to xyproto/simpleredis, and the database backends
// are interchangeable, by using the xyproto/pinterface package.
package simplebolt

import (
	"encoding/binary"
	"errors"
	"strconv"
	"strings"
	"time"

	"go.etcd.io/bbolt"
)

const (
	// Version number. Stable API within major version numbers.
	Version = 5.1
)

type (
	// Database represents Bolt database
	Database bbolt.DB

	// Used for each of the datatypes
	boltBucket struct {
		db   *Database // the Bolt database
		name []byte    // the bucket name
	}

	// List is a Bolt bucket, with methods for acting like a list
	List boltBucket

	// Set is a Bolt bucket, with methods for acting like a set, only allowing unique keys
	Set boltBucket

	// HashMap is a Bolt bucket, with methods for acting like a hash map (with an ID and then key=>value)
	HashMap boltBucket

	// KeyValue is a Bolt bucket, with methods for acting like a key=>value store
	KeyValue boltBucket
)

var (
	// ErrBucketNotFound may be returned if a no Bolt bucket was found
	ErrBucketNotFound = errors.New("Bucket not found")

	// ErrKeyNotFound will be returned if the key was not found in a HashMap or KeyValue struct
	ErrKeyNotFound = errors.New("Key not found")

	// ErrDoesNotExist will be returned if an element was not found. Used in List, Set, HashMap and KeyValue.
	ErrDoesNotExist = errors.New("Does not exist")

	// ErrExistsInSet is only returned if an element is added to a Set, but it already exists
	ErrExistsInSet = errors.New("Element already exists in set")

	// ErrInvalidID is only returned if adding an element to a HashMap that contains a colon (:)
	ErrInvalidID = errors.New("Element ID can not contain \":\"")

	// errFoundIt is only used internally, for breaking out of Bolt DB style for-loops
	errFoundIt = errors.New("Found it")

	// errReachedEnd is used internally by traversing methods to indicate that the
	// end of the data structure has been reached.
	errReachedEnd = errors.New("Reached end of data structure")
)

/* --- Database functions --- */

// New creates a new Bolt database struct, using the given file or creating a new file, as needed
func New(filename string) (*Database, error) {
	// Use a timeout, in case the database file is already in use
	db, err := bbolt.Open(filename, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	return (*Database)(db), nil
}

// Close the database
func (db *Database) Close() {
	(*bbolt.DB)(db).Close()
}

// Path returns the full path to the database file
func (db *Database) Path() string {
	return (*bbolt.DB)(db).Path()
}

// Ping the database (only for fulfilling the pinterface.IHost interface)
func (db *Database) Ping() error {
	// Always O.K.
	return nil
}

/* --- List functions --- */

// NewList loads or creates a new List struct, with the given ID
func NewList(db *Database, id string) (*List, error) {
	name := []byte(id)
	if err := (*bbolt.DB)(db).Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(name); err != nil {
			return errors.New("Could not create bucket: " + err.Error())
		}
		return nil // Return from Update function
	}); err != nil {
		return nil, err
	}
	// Success
	return &List{db, name}, nil
}

// Add an element to the list
func (l *List) Add(value string) error {
	if l.name == nil {
		return ErrDoesNotExist
	}
	return (*bbolt.DB)(l.db).Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(l.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		n, err := bucket.NextSequence()
		if err != nil {
			return err
		}
		return bucket.Put(byteID(n), []byte(value))
	})
}

// All returns all elements in the list
func (l *List) All() ([]string, error) {
	var results []string
	if l.name == nil {
		return nil, ErrDoesNotExist
	}
	err := (*bbolt.DB)(l.db).View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(l.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		return bucket.ForEach(func(_, value []byte) error {
			results = append(results, string(value))
			return nil // Continue ForEach
		})
	})
	return results, err
}

// Last will return the last element of a list
func (l *List) Last() (string, error) {
	var result string
	if l.name == nil {
		return "", ErrDoesNotExist
	}
	err := (*bbolt.DB)(l.db).View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(l.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		cursor := bucket.Cursor()
		// Ignore the key
		_, value := cursor.Last()
		result = string(value)
		return nil // Return from View function
	})
	return result, err
}

// LastN will return the last N elements of a list
func (l *List) LastN(n int) ([]string, error) {
	var results []string
	if l.name == nil {
		return nil, ErrDoesNotExist
	}
	err := (*bbolt.DB)(l.db).View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(l.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		c := bucket.Cursor()
		sizeBytes, _ := c.Last()
		size := binary.BigEndian.Uint64(sizeBytes)
		if size < uint64(n) {
			return errors.New("Too few items in list")
		}
		// Ok, fetch the n last items. startPos is counting from (size - n)+1.
		// +1 because Seek() moves to a specific key.
		// e.g. if the size of the list is, say, 50, and we want the last 4
		// elements, say, from 47 to 50 inclusive, then the calculation would be like this:
		// size = 50
		// n = 4
		// startPos = size - n // startPos = 46
		// startPos += 1 // startPos = 47
		startPos := byteID(size - uint64(n) + 1)
		for key, value := c.Seek(startPos); key != nil; key, value = c.Next() {
			results = append(results, string(value))
		}
		return nil // Return from View function
	})
	return results, err
}

// Remove this list
func (l *List) Remove() error {
	err := (*bbolt.DB)(l.db).Update(func(tx *bbolt.Tx) error {
		return tx.DeleteBucket(l.name)
	})
	// Mark as removed by setting the name to nil
	l.name = nil
	return err
}

// Clear will remove all elements from this list
func (l *List) Clear() error {
	if l.name == nil {
		return ErrDoesNotExist
	}
	return (*bbolt.DB)(l.db).Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(l.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		return bucket.ForEach(func(key, _ []byte) error {
			return bucket.Delete(key)
		})
	})
}

/* --- Set functions --- */

// NewSet loads or creates a new Set struct, with the given ID
func NewSet(db *Database, id string) (*Set, error) {
	name := []byte(id)
	if err := (*bbolt.DB)(db).Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(name); err != nil {
			return errors.New("Could not create bucket: " + err.Error())
		}
		return nil // Return from Update function
	}); err != nil {
		return nil, err
	}
	// Success
	return &Set{db, name}, nil
}

// Add an element to the set
func (s *Set) Add(value string) error {
	if s.name == nil {
		return ErrDoesNotExist
	}
	exists, err := s.Has(value)
	if err != nil {
		return err
	}
	if exists {
		return ErrExistsInSet
	}
	return (*bbolt.DB)(s.db).Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(s.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		n, err := bucket.NextSequence()
		if err != nil {
			return err
		}
		return bucket.Put(byteID(n), []byte(value))
	})
}

// Has will check if a given value is in the set
func (s *Set) Has(value string) (bool, error) {
	var exists bool
	if s.name == nil {
		return false, ErrDoesNotExist
	}
	err := (*bbolt.DB)(s.db).View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(s.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		bucket.ForEach(func(_, byteValue []byte) error {
			if value == string(byteValue) {
				exists = true
				return errFoundIt // break the ForEach by returning an error
			}
			return nil // Continue ForEach
		})
		return nil // Return from View function
	})
	return exists, err
}

// All returns all elements in the set
func (s *Set) All() ([]string, error) {
	var values []string
	if s.name == nil {
		return nil, ErrDoesNotExist
	}
	err := (*bbolt.DB)(s.db).View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(s.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		return bucket.ForEach(func(_, value []byte) error {
			values = append(values, string(value))
			return nil // Return from ForEach function
		})
	})
	return values, err
}

// Del will remove an element from the set
func (s *Set) Del(value string) error {
	if s.name == nil {
		return ErrDoesNotExist
	}
	return (*bbolt.DB)(s.db).Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(s.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		var foundKey []byte
		bucket.ForEach(func(byteKey, byteValue []byte) error {
			if value == string(byteValue) {
				foundKey = byteKey
				return errFoundIt // break the ForEach by returning an error
			}
			return nil // Continue ForEach
		})
		return bucket.Delete([]byte(foundKey))
	})
}

// Remove this set
func (s *Set) Remove() error {
	err := (*bbolt.DB)(s.db).Update(func(tx *bbolt.Tx) error {
		return tx.DeleteBucket(s.name)
	})
	// Mark as removed by setting the name to nil
	s.name = nil
	return err
}

// Clear will remove all elements from this set
func (s *Set) Clear() error {
	if s.name == nil {
		return ErrDoesNotExist
	}
	return (*bbolt.DB)(s.db).Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(s.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		return bucket.ForEach(func(key, _ []byte) error {
			return bucket.Delete(key)
		})
	})
}

/* --- HashMap functions --- */

// NewHashMap loads or creates a new HashMap struct, with the given ID
func NewHashMap(db *Database, id string) (*HashMap, error) {
	name := []byte(id)
	if err := (*bbolt.DB)(db).Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(name); err != nil {
			return errors.New("Could not create bucket: " + err.Error())
		}
		return nil // Return from Update function
	}); err != nil {
		return nil, err
	}
	// Success
	return &HashMap{db, name}, nil
}

// Set a value in a hashmap given the element id (for instance a user id) and the key (for instance "password")
func (h *HashMap) Set(elementid, key, value string) error {
	if h.name == nil {
		return ErrDoesNotExist
	}
	if strings.Contains(elementid, ":") {
		return ErrInvalidID
	}
	return (*bbolt.DB)(h.db).Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(h.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		// Store the key and value
		return bucket.Put([]byte(elementid+":"+key), []byte(value))
	})
}

// All returns all ID's, for all hash elements
func (h *HashMap) All() ([]string, error) {
	var results []string
	if h.name == nil {
		return nil, ErrDoesNotExist
	}
	err := (*bbolt.DB)(h.db).View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(h.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		return bucket.ForEach(func(byteKey, _ []byte) error {
			combinedKey := string(byteKey)
			if strings.Contains(combinedKey, ":") {
				fields := strings.SplitN(combinedKey, ":", 2)
				for _, result := range results {
					if result == fields[0] {
						// Result already exists, continue
						return nil // Continue ForEach
					}
				}
				// Store the new result
				results = append(results, string(fields[0]))
			}
			return nil // Continue ForEach
		})
	})
	return results, err
}

// Get a value from a hashmap given the element id (for instance a user id) and the key (for instance "password")
func (h *HashMap) Get(elementid, key string) (string, error) {
	var val string
	if h.name == nil {
		return "", ErrDoesNotExist
	}
	err := (*bbolt.DB)(h.db).View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(h.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		byteval := bucket.Get([]byte(elementid + ":" + key))
		if byteval == nil {
			return ErrKeyNotFound
		}
		val = string(byteval)
		return nil // Return from View function
	})
	return val, err
}

// Has will check if a given elementid + key is in the hash map
func (h *HashMap) Has(elementid, key string) (bool, error) {
	var found bool
	if h.name == nil {
		return false, ErrDoesNotExist
	}
	err := (*bbolt.DB)(h.db).View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(h.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		byteval := bucket.Get([]byte(elementid + ":" + key))
		if byteval != nil {
			found = true
		}
		return nil // Return from View function
	})
	return found, err
}

// Keys returns all names of all keys of a given owner.
func (h *HashMap) Keys(owner string) ([]string, error) {
	var props []string
	err := (*bbolt.DB)(h.db).View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(h.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		// Loop through the keys
		return bucket.ForEach(func(byteKey, _ []byte) error {
			combinedKey := string(byteKey)
			if strings.HasPrefix(combinedKey, owner+":") {
				// Store the right side of the bucket key, after ":"
				fields := strings.SplitN(combinedKey, ":", 2)
				props = append(props, string(fields[1]))
			}
			return nil // Continue ForEach
		})
	})
	return props, err
}

// Exists will check if a given elementid exists as a hash map at all
func (h *HashMap) Exists(elementid string) (bool, error) {
	var found bool
	if h.name == nil {
		return false, ErrDoesNotExist
	}
	err := (*bbolt.DB)(h.db).View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(h.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		bucket.ForEach(func(byteKey, byteValue []byte) error {
			combinedKey := string(byteKey)
			if strings.Contains(combinedKey, ":") {
				fields := strings.SplitN(combinedKey, ":", 2)
				if fields[0] == elementid {
					found = true
					return errFoundIt
				}
			}
			return nil // Continue ForEach
		})
		return nil // Return from View function
	})
	return found, err
}

// DelKey will remove a key for an entry in a hashmap (for instance the email field for a user)
func (h *HashMap) DelKey(elementid, key string) error {
	if h.name == nil {
		return ErrDoesNotExist
	}
	return (*bbolt.DB)(h.db).Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(h.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		return bucket.Delete([]byte(elementid + ":" + key))
	})
}

// Del will remove an element (for instance a user)
func (h *HashMap) Del(elementid string) error {
	if h.name == nil {
		return ErrDoesNotExist
	}
	// Remove the keys starting with elementid + ":"
	return (*bbolt.DB)(h.db).Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(h.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		return bucket.ForEach(func(byteKey, byteValue []byte) error {
			combinedKey := string(byteKey)
			if strings.Contains(combinedKey, ":") {
				fields := strings.SplitN(combinedKey, ":", 2)
				if fields[0] == elementid {
					return bucket.Delete([]byte(combinedKey))
				}
			}
			return nil // Continue ForEach
		})
	})
}

// Remove this hashmap
func (h *HashMap) Remove() error {
	err := (*bbolt.DB)(h.db).Update(func(tx *bbolt.Tx) error {
		return tx.DeleteBucket(h.name)
	})
	// Mark as removed by setting the name to nil
	h.name = nil
	return err
}

// Clear will remove all elements from this hash map
func (h *HashMap) Clear() error {
	if h.name == nil {
		return ErrDoesNotExist
	}
	return (*bbolt.DB)(h.db).Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(h.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		return bucket.ForEach(func(key, _ []byte) error {
			return bucket.Delete(key)
		})
	})
}

/* --- KeyValue functions --- */

// NewKeyValue loads or creates a new KeyValue struct, with the given ID
func NewKeyValue(db *Database, id string) (*KeyValue, error) {
	name := []byte(id)
	if err := (*bbolt.DB)(db).Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(name); err != nil {
			return errors.New("Could not create bucket: " + err.Error())
		}
		return nil // Return from Update function
	}); err != nil {
		return nil, err
	}
	return &KeyValue{db, name}, nil
}

// Set a key and value
func (kv *KeyValue) Set(key, value string) error {
	if kv.name == nil {
		return ErrDoesNotExist
	}
	return (*bbolt.DB)(kv.db).Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(kv.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		return bucket.Put([]byte(key), []byte(value))
	})
}

// Get a value given a key
// Returns an error if the key was not found
func (kv *KeyValue) Get(key string) (string, error) {
	var val string
	if kv.name == nil {
		return "", ErrDoesNotExist
	}
	err := (*bbolt.DB)(kv.db).View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(kv.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		byteval := bucket.Get([]byte(key))
		if byteval == nil {
			return ErrKeyNotFound
		}
		val = string(byteval)
		return nil // Return from View function
	})
	return val, err
}

// Del will remove a key
func (kv *KeyValue) Del(key string) error {
	if kv.name == nil {
		return ErrDoesNotExist
	}
	return (*bbolt.DB)(kv.db).Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(kv.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		return bucket.Delete([]byte(key))
	})
}

// Inc will increase the value of a key, returns the new value
// Returns an empty string if there were errors,
// or "0" if the key does not already exist.
func (kv *KeyValue) Inc(key string) (string, error) {
	var val string
	if kv.name == nil {
		kv.name = []byte(key)
	}
	err := (*bbolt.DB)(kv.db).Update(func(tx *bbolt.Tx) (err error) {
		// The numeric value
		num := 0
		// Get the string value
		bucket := tx.Bucket(kv.name)
		if bucket == nil {
			// Create the bucket if it does not already exist
			bucket, err = tx.CreateBucketIfNotExists(kv.name)
			if err != nil {
				return errors.New("Could not create bucket: " + err.Error())
			}
		} else {
			val := string(bucket.Get([]byte(key)))
			if converted, err := strconv.Atoi(val); err == nil {
				// Conversion successful
				num = converted
			}
		}
		// Num is now either 0 or the previous numeric value
		num++
		// Convert the new value to a string and save it
		val = strconv.Itoa(num)
		err = bucket.Put([]byte(key), []byte(val))
		return err
	})
	return val, err
}

// Remove this key/value
func (kv *KeyValue) Remove() error {
	err := (*bbolt.DB)(kv.db).Update(func(tx *bbolt.Tx) error {
		return tx.DeleteBucket(kv.name)
	})
	// Mark as removed by setting the name to nil
	kv.name = nil
	return err
}

// Clear will remove all elements from this key/value
func (kv *KeyValue) Clear() error {
	if kv.name == nil {
		return ErrDoesNotExist
	}
	return (*bbolt.DB)(kv.db).Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(kv.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		return bucket.ForEach(func(key, _ []byte) error {
			return bucket.Delete(key)
		})
	})
}

/* --- Utility functions --- */

// Create a byte slice from an uint64
func byteID(x uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, x)
	return b
}
