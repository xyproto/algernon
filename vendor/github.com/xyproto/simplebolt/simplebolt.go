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

	"github.com/coreos/bbolt"
)

const (
	// Version number. Stable API within major version numbers.
	Version = 3.2
)

type (
	// Database represents Bolt database
	Database bolt.DB

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

	// ErrFoundIt is only used internally, for breaking out of Bolt DB style for-loops
	ErrFoundIt = errors.New("Found it")
)

/* --- Database functions --- */

// New creates a new Bolt database struct, using the given file or creating a new file, as needed
func New(filename string) (*Database, error) {
	// Use a timeout, in case the database file is already in use
	db, err := bolt.Open(filename, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	return (*Database)(db), nil
}

// Close the database
func (db *Database) Close() {
	(*bolt.DB)(db).Close()
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
	if err := (*bolt.DB)(db).Update(func(tx *bolt.Tx) error {
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
	return (*bolt.DB)(l.db).Update(func(tx *bolt.Tx) error {
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
func (l *List) All() (results []string, err error) {
	if l.name == nil {
		return nil, ErrDoesNotExist
	}
	return results, (*bolt.DB)(l.db).View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(l.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		return bucket.ForEach(func(_, value []byte) error {
			results = append(results, string(value))
			return nil // Continue ForEach
		})
	})
}

// GetAll is deprecated, please use .All() instead
func (l *List) GetAll() ([]string, error) {
	return l.All()
}

// Get the last element of a list
func (l *List) Last() (result string, err error) {
	if l.name == nil {
		return "", ErrDoesNotExist
	}
	return result, (*bolt.DB)(l.db).View(func(tx *bolt.Tx) error {
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
}

// Deprecated
func (l *List) GetLast() (string, error) {
	return l.Last()
}

// Get the last N elements of a list
func (l *List) LastN(n int) (results []string, err error) {
	if l.name == nil {
		return nil, ErrDoesNotExist
	}
	return results, (*bolt.DB)(l.db).View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(l.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		var size int64
		bucket.ForEach(func(_, _ []byte) error {
			size++
			return nil // Continue ForEach
		})
		if size < int64(n) {
			return errors.New("Too few items in list")
		}
		// Ok, fetch the n last items. startPos is counting from 0.
		var (
			startPos = size - int64(n)
			i        int64
		)
		bucket.ForEach(func(_, value []byte) error {
			if i >= startPos {
				results = append(results, string(value))
			}
			i++
			return nil // Continue ForEach
		})
		return nil // Return from View function
	})
}

// Deprecated
func (l *List) GetLastN(n int) ([]string, error) {
	return l.LastN(n)
}

// Remove this list
func (l *List) Remove() error {
	err := (*bolt.DB)(l.db).Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket(l.name)
	})
	// Mark as removed by setting the name to nil
	l.name = nil
	return err
}

// Remove all elements from this list
func (l *List) Clear() error {
	if l.name == nil {
		return ErrDoesNotExist
	}
	return (*bolt.DB)(l.db).Update(func(tx *bolt.Tx) error {
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
	if err := (*bolt.DB)(db).Update(func(tx *bolt.Tx) error {
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
	return (*bolt.DB)(s.db).Update(func(tx *bolt.Tx) error {
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

// Check if a given value is in the set
func (s *Set) Has(value string) (exists bool, err error) {
	if s.name == nil {
		return false, ErrDoesNotExist
	}
	return exists, (*bolt.DB)(s.db).View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		bucket.ForEach(func(_, byteValue []byte) error {
			if value == string(byteValue) {
				exists = true
				return ErrFoundIt // break the ForEach by returning an error
			}
			return nil // Continue ForEach
		})
		return nil // Return from View function
	})
}

// All returns all elements in the set
func (s *Set) All() (values []string, err error) {
	if s.name == nil {
		return nil, ErrDoesNotExist
	}
	return values, (*bolt.DB)(s.db).View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		return bucket.ForEach(func(_, value []byte) error {
			values = append(values, string(value))
			return nil // Return from ForEach function
		})
	})
}

// GetAll is deprecated, please use .All() instead
func (s *Set) GetAll() ([]string, error) {
	return s.All()
}

// Remove an element from the set
func (s *Set) Del(value string) error {
	if s.name == nil {
		return ErrDoesNotExist
	}
	return (*bolt.DB)(s.db).Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		var foundKey []byte
		bucket.ForEach(func(byteKey, byteValue []byte) error {
			if value == string(byteValue) {
				foundKey = byteKey
				return ErrFoundIt // break the ForEach by returning an error
			}
			return nil // Continue ForEach
		})
		return bucket.Delete([]byte(foundKey))
	})
}

// Remove this set
func (s *Set) Remove() error {
	err := (*bolt.DB)(s.db).Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket(s.name)
	})
	// Mark as removed by setting the name to nil
	s.name = nil
	return err
}

// Remove all elements from this set
func (s *Set) Clear() error {
	if s.name == nil {
		return ErrDoesNotExist
	}
	return (*bolt.DB)(s.db).Update(func(tx *bolt.Tx) error {
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
	if err := (*bolt.DB)(db).Update(func(tx *bolt.Tx) error {
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
func (h *HashMap) Set(elementid, key, value string) (err error) {
	if h.name == nil {
		return ErrDoesNotExist
	}
	if strings.Contains(elementid, ":") {
		return ErrInvalidID
	}
	return (*bolt.DB)(h.db).Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(h.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		// Store the key and value
		return bucket.Put([]byte(elementid+":"+key), []byte(value))
	})
}

// All returns all ID's, for all hash elements
func (h *HashMap) All() (results []string, err error) {
	if h.name == nil {
		return nil, ErrDoesNotExist
	}
	return results, (*bolt.DB)(h.db).View(func(tx *bolt.Tx) error {
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
}

// GetAll is deprecated, please use .All() instead
func (h *HashMap) GetAll() ([]string, error) {
	return h.All()
}

// Get a value from a hashmap given the element id (for instance a user id) and the key (for instance "password")
func (h *HashMap) Get(elementid, key string) (val string, err error) {
	if h.name == nil {
		return "", ErrDoesNotExist
	}
	err = (*bolt.DB)(h.db).View(func(tx *bolt.Tx) error {
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
	return
}

// Check if a given elementid + key is in the hash map
func (h *HashMap) Has(elementid, key string) (found bool, err error) {
	if h.name == nil {
		return false, ErrDoesNotExist
	}
	return found, (*bolt.DB)(h.db).View(func(tx *bolt.Tx) error {
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
}

// Keys returns all names of all keys of a given owner.
func (h *HashMap) Keys(owner string) ([]string, error) {
	var props []string
	return props, (*bolt.DB)(h.db).View(func(tx *bolt.Tx) error {
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
}

// Check if a given elementid exists as a hash map at all
func (h *HashMap) Exists(elementid string) (found bool, err error) {
	if h.name == nil {
		return false, ErrDoesNotExist
	}
	return found, (*bolt.DB)(h.db).View(func(tx *bolt.Tx) error {
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
					return ErrFoundIt
				}
			}
			return nil // Continue ForEach
		})
		return nil // Return from View function
	})
}

// Remove a key for an entry in a hashmap (for instance the email field for a user)
func (h *HashMap) DelKey(elementid, key string) error {
	if h.name == nil {
		return ErrDoesNotExist
	}
	return (*bolt.DB)(h.db).Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(h.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		return bucket.Delete([]byte(elementid + ":" + key))
	})
}

// Remove an element (for instance a user)
func (h *HashMap) Del(elementid string) error {
	if h.name == nil {
		return ErrDoesNotExist
	}
	// Remove the keys starting with elementid + ":"
	return (*bolt.DB)(h.db).Update(func(tx *bolt.Tx) error {
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
	err := (*bolt.DB)(h.db).Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket(h.name)
	})
	// Mark as removed by setting the name to nil
	h.name = nil
	return err
}

// Remove all elements from this hash map
func (h *HashMap) Clear() error {
	if h.name == nil {
		return ErrDoesNotExist
	}
	return (*bolt.DB)(h.db).Update(func(tx *bolt.Tx) error {
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
	if err := (*bolt.DB)(db).Update(func(tx *bolt.Tx) error {
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
	return (*bolt.DB)(kv.db).Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(kv.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		return bucket.Put([]byte(key), []byte(value))
	})
}

// Get a value given a key
// Returns an error if the key was not found
func (kv *KeyValue) Get(key string) (val string, err error) {
	if kv.name == nil {
		return "", ErrDoesNotExist
	}
	err = (*bolt.DB)(kv.db).View(func(tx *bolt.Tx) error {
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
	return
}

// Remove a key
func (kv *KeyValue) Del(key string) error {
	if kv.name == nil {
		return ErrDoesNotExist
	}
	return (*bolt.DB)(kv.db).Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(kv.name)
		if bucket == nil {
			return ErrBucketNotFound
		}
		return bucket.Delete([]byte(key))
	})
}

// Increase the value of a key, returns the new value
// Returns an empty string if there were errors,
// or "0" if the key does not already exist.
func (kv *KeyValue) Inc(key string) (val string, err error) {
	if kv.name == nil {
		kv.name = []byte(key)
	}
	return val, (*bolt.DB)(kv.db).Update(func(tx *bolt.Tx) error {
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
}

// Remove this key/value
func (kv *KeyValue) Remove() error {
	err := (*bolt.DB)(kv.db).Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket(kv.name)
	})
	// Mark as removed by setting the name to nil
	kv.name = nil
	return err
}

// Remove all elements from this key/value
func (kv *KeyValue) Clear() error {
	if kv.name == nil {
		return ErrDoesNotExist
	}
	return (*bolt.DB)(kv.db).Update(func(tx *bolt.Tx) error {
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
