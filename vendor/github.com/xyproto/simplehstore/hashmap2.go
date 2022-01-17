package simplehstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/lib/pq"
)

// HashMap2 contains a KeyValue struct and a dbDatastructure.
// Each value is a JSON data blob and can contains sub-keys.
type HashMap2 struct {
	dbDatastructure        // KeyValue is .host *Host + .table string
	seenPropTable   string // Set of all encountered property keys
}

// A string that is unlikely to appear in a key
const fieldSep = "¤"

// NewHashMap2 creates a new HashMap2 struct
func NewHashMap2(host *Host, name string) (*HashMap2, error) {
	var hm2 HashMap2
	// kv is a KeyValue (HSTORE) table of all properties (key = owner_ID + "¤" + property_key)
	kv, err := NewKeyValue(host, name+"_properties_HSTORE_map")
	if err != nil {
		return nil, err
	}
	// seenPropSet is a set of all encountered property keys
	seenPropSet, err := NewSet(host, name+"_encountered_property_keys")
	if err != nil {
		return nil, err
	}
	hm2.host = host
	hm2.table = kv.table
	hm2.seenPropTable = seenPropSet.table
	return &hm2, nil
}

// keyValue returns the *KeyValue of properties for this HashMap2
func (hm2 *HashMap2) keyValue() *KeyValue {
	return &KeyValue{hm2.host, hm2.table}
}

// propSet returns the property *Set for this HashMap2
func (hm2 *HashMap2) propSet() *Set {
	return &Set{hm2.host, hm2.seenPropTable}
}

// Set a value in a hashmap given the element id (for instance a user id) and the key (for instance "password")
func (hm2 *HashMap2) Set(owner, key, value string) error {
	return hm2.SetMap(owner, map[string]string{key: value})
}

// updatePropWithTransaction will set a value in a hashmap given the element id (for instance a user id) and the key (for instance "password")
// Note that the database can not be empty when calling this! The HSTORE must be initialized first, possibly with an INSERT!
func (hm2 *HashMap2) updatePropWithTransaction(ctx context.Context, transaction *sql.Tx, owner, key, value string, checkForFieldSep bool) error {
	if checkForFieldSep {
		if strings.Contains(owner, fieldSep) {
			return fmt.Errorf("owner can not contain %s", fieldSep)
		}
		if strings.Contains(key, fieldSep) {
			return fmt.Errorf("key can not contain %s", fieldSep)
		}
	}
	// Add the key to the property set, without using a transaction
	if err := hm2.propSet().Add(key); err != nil {
		return err
	}
	// Set a key + value for this "owner¤key"
	kv := hm2.keyValue()
	if !kv.host.rawUTF8 {
		Encode(&value)
	}
	encodedValue := value
	_, err := kv.updateWithTransaction(ctx, transaction, owner+fieldSep+key, encodedValue)
	return err
}

// insertPropWithTransaction will set a value in a hashmap given the element id (for instance a user id) and the key (for instance "password")
// Note that the database can not be empty when calling this! The HSTORE must be initialized first, possibly with an INSERT!
func (hm2 *HashMap2) insertPropWithTransaction(ctx context.Context, transaction *sql.Tx, owner, key, value string, checkForFieldSep bool) error {
	if checkForFieldSep {
		if strings.Contains(owner, fieldSep) {
			return fmt.Errorf("owner can not contain %s", fieldSep)
		}
		if strings.Contains(key, fieldSep) {
			return fmt.Errorf("key can not contain %s", fieldSep)
		}
	}
	// Add the key to the property set, without using a transaction
	if err := hm2.propSet().Add(key); err != nil {
		return err
	}
	// Set a key + value for this "owner¤key"
	kv := hm2.keyValue()
	if !kv.host.rawUTF8 {
		Encode(&value)
	}
	encodedValue := value
	_, err := kv.insertWithTransaction(ctx, transaction, owner+fieldSep+key, encodedValue)
	return err
}

// SetMap will set many keys/values, in a single transaction
func (hm2 *HashMap2) SetMap(owner string, m map[string]string) error {
	checkForFieldSep := true

	// Get all properties
	propset := hm2.propSet()
	allProperties, err := propset.All()
	if err != nil {
		return err
	}

	isEmpty, err := hm2.keyValue().Empty()
	if err != nil {
		return err
	}

	// Use a context and a transaction to bundle queries
	ctx := context.Background()
	transaction, err := hm2.host.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	insertedKey := ""
	if isEmpty { // Insert just one key, to initialize the HSTORE value
		// Prepare the changes
		for k, v := range m {
			if err := hm2.insertPropWithTransaction(ctx, transaction, owner, k, v, checkForFieldSep); err != nil {
				transaction.Rollback()
				return err
			}
			if !hasS(allProperties, k) {
				if err := propset.Add(k); err != nil {
					transaction.Rollback()
					return err
				}
			}
			insertedKey = k
			break
		}
	}
	// Prepare the changes
	for k, v := range m { // Update the rest
		if k == insertedKey {
			continue
		}
		if err := hm2.updatePropWithTransaction(ctx, transaction, owner, k, v, checkForFieldSep); err != nil {
			transaction.Rollback()
			return err
		}
		if !hasS(allProperties, k) {
			if err := propset.Add(k); err != nil {
				transaction.Rollback()
				return err
			}
		}
	}

	return transaction.Commit()
}

// SetLargeMap will add many owners+keys/values, in a single transaction, without checking if they already exists.
// It also does not check if the keys or property keys contains fieldSep (¤) or not, for performance.
// These must all be brand new "usernames" (the first key), and not be in the existing hm2.OwnerSet().
// This function has good performance, but must be used carefully.
func (hm2 *HashMap2) SetLargeMap(allProperties map[string]map[string]string) error {

	// First get the KeyValue and Set structures that will be used
	kv := hm2.keyValue()
	propSet := hm2.propSet()

	// All seen properties
	props, err := propSet.All()
	if err != nil {
		return err
	}

	// Check if the KeyValue table is empty or not
	isEmpty, err := kv.Empty()
	if err != nil {
		return err
	}

	// Find new properties in the allProperties map
	var newProps []string
	for owner := range allProperties {
		// Find all unique properties
		for k := range allProperties[owner] {
			if !hasS(props, k) && !hasS(newProps, k) {
				newProps = append(newProps, k)
			}
		}
	}

	ctx := context.Background()

	if Verbose {
		fmt.Println("Starting transaction")
	}

	// Create a new transaction
	transaction, err := hm2.host.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Store the new properties
	for _, prop := range newProps {
		if Verbose {
			fmt.Printf("ADDING %s\n", prop)
		}
		if err := propSet.addWithTransactionNoCheck(ctx, transaction, prop); err != nil {
			return err
		}
	}

	var (
		sb                     strings.Builder
		beyondFirst            bool
		firstO, firstK, firstV string
	)

	// Build a long key+value string
	for owner, propMap := range allProperties {
		for k, v := range propMap {
			if firstO == "" {
				firstO = owner
				firstK = k
				firstV = v
			}
			if beyondFirst {
				sb.WriteString(",")
			} else {
				beyondFirst = true
			}
			if !kv.host.rawUTF8 {
				Encode(&v)
			}
			sb.WriteString("\"" + owner + fieldSep + k + "\"=>\"" + v + "\"")
		}
	}

	// Initialize the HSTORE, if needed
	if isEmpty && firstO != "" && firstK != "" && firstV != "" {
		encodedValue := firstV
		if !kv.host.rawUTF8 {
			Encode(&encodedValue)
		}
		kv.insert(firstO+fieldSep+firstK, encodedValue)
	}

	// Try setting+updating all values, in a transaction
	query := fmt.Sprintf("UPDATE %s SET attr = attr || '%s' :: hstore", pq.QuoteIdentifier(kvPrefix+kv.table), escapeSingleQuotes(sb.String()))
	if Verbose {
		fmt.Println(query)
	}
	result, err := transaction.ExecContext(ctx, query)
	if Verbose {
		log.Println("Updated row in: "+kv.table+" err? ", err)
	}
	if result == nil {
		transaction.Rollback()
		return fmt.Errorf("keyValue updateWithTransaction: no result when updating with %s", sb.String())
	}
	_, err = result.RowsAffected()
	if err != nil {
		transaction.Rollback()
		return err
	}

	if Verbose {
		fmt.Println("Committing transaction")
	}
	if err := transaction.Commit(); err != nil {
		return err
	}

	fmt.Println("Transaction complete")

	return nil // success
}

// Get a value.
// Returns: value, error
// If a value was not found, an empty string is returned.
func (hm2 *HashMap2) Get(owner, key string) (string, error) {
	return hm2.keyValue().Get(owner + fieldSep + key)
}

// GetMap can retrieve multiple values in one transaction
func (hm2 *HashMap2) GetMap(owner string, keys []string) (map[string]string, error) {
	results := make(map[string]string)

	// Use a context and a transaction to bundle queries
	ctx := context.Background()
	transaction, err := hm2.host.db.BeginTx(ctx, nil)
	if err != nil {
		return results, err
	}

	for _, key := range keys {
		s, err := hm2.keyValue().getWithTransaction(ctx, transaction, owner+fieldSep+key)
		if err != nil {
			transaction.Rollback()
			return results, err
		}
		results[key] = s
	}

	transaction.Commit()
	return results, nil
}

// Has checks if a given owner + key exists in the hash map
func (hm2 *HashMap2) Has(owner, key string) (bool, error) {
	s, err := hm2.keyValue().Get(owner + fieldSep + key)
	if err != nil {
		if noResult(err) {
			// Not an actual error, just got no results
			return false, nil
		}
		// An actual error
		return false, err
	}
	// No error, got a result
	if s == "" {
		return false, nil
	}
	return true, nil
}

// Exists checks if a given owner exists as a hash map at all.
func (hm2 *HashMap2) Exists(owner string) (bool, error) {
	kv := hm2.keyValue()
	query := fmt.Sprintf("SELECT SUBSTRING(skeys,'(.*)%s') FROM (SELECT skeys(attr) FROM %s) AS temp WHERE skeys LIKE '%s%s%%' LIMIT 1",
		fieldSep,
		pq.QuoteIdentifier(kvPrefix+kv.table),
		owner,
		fieldSep,
	)
	rows, err := kv.host.db.Query(query)
	if err != nil {
		return false, err
	}
	if rows == nil {
		return false, errors.New("hashMap2 Exists returned no rows for owner " + owner)
	}
	defer rows.Close()
	var scanValue sql.NullString
	// Get the value. Should not loop more than once.
	counter := 0
	for rows.Next() {
		err = rows.Scan(&scanValue)
		if err != nil {
			// No rows
			return false, err
		}
		counter++
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return counter > 0, nil
}

// AllWhere returns all owner ID's that has a property where key == value
func (hm2 *HashMap2) AllWhere(key, value string) ([]string, error) {
	kv := hm2.keyValue()
	if !kv.host.rawUTF8 {
		Encode(&value)
	}
	query := fmt.Sprintf("SELECT SUBSTRING(skeys,'(.*)%s') FROM (SELECT skeys(attr), svals(attr) FROM %s) AS temp WHERE skeys LIKE '%%%s%s' AND svals = '%s'",
		fieldSep,
		pq.QuoteIdentifier(kvPrefix+kv.table),
		fieldSep,
		key,
		value,
	)
	rows, err := kv.host.db.Query(query)
	if err != nil {
		return []string{}, err
	}
	if rows == nil {
		return []string{}, ErrNoAvailableValues
	}
	defer rows.Close()
	var v sql.NullString
	var values []string
	for rows.Next() {
		err = rows.Scan(&v)
		vs := v.String
		values = append(values, vs)
		if err != nil {
			return values, err
		}
	}
	err = rows.Err()
	return values, err
}

// AllPossibleKeys returns all encountered keys for all owners
func (hm2 *HashMap2) AllPossibleKeys() ([]string, error) {
	return hm2.propSet().All()
}

// Keys loops through absolutely all owners and all properties in the database
// and returns all found keys.
func (hm2 *HashMap2) Keys(owner string) ([]string, error) {
	allProps, err := hm2.propSet().All()
	if err != nil {
		return []string{}, err
	}
	// TODO: Improve the performance of this by using SQL instead of looping
	allKeys := []string{}
	for _, key := range allProps {
		if found, err := hm2.Has(owner, key); err == nil && found {
			allKeys = append(allKeys, key)
		}
	}
	return allKeys, nil
}

// All returns all owner ID's
func (hm2 *HashMap2) All() ([]string, error) {
	foundOwners := make(map[string]bool)
	allOwnersAndKeys, err := hm2.keyValue().All()
	if err != nil {
		return []string{}, err
	}
	for _, ownerAndKey := range allOwnersAndKeys {
		if pos := strings.Index(ownerAndKey, fieldSep); pos != -1 {
			owner := ownerAndKey[:pos]
			if _, has := foundOwners[owner]; !has {
				foundOwners[owner] = true
			}
		}
	}
	keys := make([]string, len(foundOwners))
	i := 0
	for k := range foundOwners {
		keys[i] = k
		i++
	}
	return keys, nil
}

// Count counts the number of owners for hash map elements
func (hm2 *HashMap2) Count() (int64, error) {
	a, err := hm2.All()
	if err != nil {
		return 0, err
	}
	return int64(len(a)), nil
	// return hm2.KeyValue().Count() is not correct, since it counts all owners + fieldSep + keys
}

// DelKey removes a key of an owner in a hashmap (for instance the email field for a user)
func (hm2 *HashMap2) DelKey(owner, key string) error {
	// The key is not removed from the set of all encountered properties
	// even if it's the last key with that name, for a performance vs storage tradeoff.
	return hm2.keyValue().Del(owner + fieldSep + key)
}

// Del removes an element (for instance a user)
func (hm2 *HashMap2) Del(owner string) error {
	allProps, err := hm2.propSet().All()
	if err != nil {
		return err
	}
	for _, key := range allProps {
		if err := hm2.keyValue().Del(owner + fieldSep + key); err != nil {
			return err
		}
	}
	return nil
}

// Remove this hashmap
func (hm2 *HashMap2) Remove() error {
	hm2.propSet().Remove()
	if err := hm2.keyValue().Remove(); err != nil {
		return fmt.Errorf("could not remove kv: %s", err)
	}
	return nil
}

// Clear the contents
func (hm2 *HashMap2) Clear() error {
	hm2.propSet().Clear()
	if err := hm2.keyValue().Clear(); err != nil {
		return err
	}
	return nil
}

// Empty checks if there are no owners+keys+values
func (hm2 *HashMap2) Empty() (bool, error) {
	return hm2.keyValue().Empty()
}
