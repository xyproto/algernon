package simpleredis

import (
	"github.com/xyproto/pinterface"
	"log"
	"strings"
	"testing"
)

var pool *ConnectionPool

func TestLocalConnection(t *testing.T) {
	if err := TestConnection(); err != nil {
		if strings.HasSuffix(err.Error(), "i/o timeout") {
			log.Println("Try the 'latency doctor' command in the redis-cli if i/o timeouts is a common problem.")
		}
		t.Errorf(err.Error())
	}
}

func TestRemoteConnection(t *testing.T) {
	if err := TestConnectionHost("foobared@ :6379"); err != nil {
		t.Errorf(err.Error())
	}
}

func TestConnectionPool(t *testing.T) {
	pool = NewConnectionPool()
}

func TestConnectionPoolHost(t *testing.T) {
	pool = NewConnectionPoolHost("localhost:6379")
}

// Tests with password "foobared" if the previous connection test
// did not result in a connection that responds to PING.
func TestConnectionPoolHostPassword(t *testing.T) {
	if pool.Ping() != nil {
		// Try connecting with the default password
		pool = NewConnectionPoolHost("foobared@localhost:6379")
	}
}

func TestList(t *testing.T) {
	const (
		listname = "abc123_test_test_test_123abc"
		testdata = "123abc"
	)
	list := NewList(pool, listname)

	// Check that the list qualifies for the IList interface
	var _ pinterface.IList = list

	list.SelectDatabase(1)
	if err := list.Add(testdata); err != nil {
		t.Errorf("Error, could not add item to list! %s", err.Error())
	}
	items, err := list.GetAll()
	if err != nil {
		t.Errorf("Error, could not retrieve list! %s", err.Error())
	}
	if len(items) != 1 {
		t.Errorf("Error, wrong list length! %v", len(items))
	}
	if (len(items) > 0) && (items[0] != testdata) {
		t.Errorf("Error, wrong list contents! %v", items)
	}
	err = list.Remove()
	if err != nil {
		t.Errorf("Error, could not remove list! %s", err.Error())
	}
}

func TestRemove(t *testing.T) {
	const (
		kvname    = "abc123_test_test_test_123abc"
		testkey   = "sdsdf234234"
		testvalue = "asdfasdf1234"
	)
	kv := NewKeyValue(pool, kvname)

	// TODO: Also do this check for ISet and IHashMap
	// Check that the key/value qualifies for the IKeyValue interface
	var _ pinterface.IKeyValue = kv

	kv.SelectDatabase(1)
	if err := kv.Set(testkey, testvalue); err != nil {
		t.Errorf("Error, could not set key and value! %s", err.Error())
	}
	if val, err := kv.Get(testkey); err != nil {
		t.Errorf("Error, could not get key! %s", err.Error())
	} else if val != testvalue {
		t.Errorf("Error, wrong value! %s != %s", val, testvalue)
	}
	kv.Remove()
	if _, err := kv.Get(testkey); err == nil {
		t.Errorf("Error, could get key! %s", err.Error())
	}
}

func TestInc(t *testing.T) {
	const (
		kvname     = "kv_234_test_test_test"
		testkey    = "key_234_test_test_test"
		testvalue0 = "9"
		testvalue1 = "10"
		testvalue2 = "1"
	)
	kv := NewKeyValue(pool, kvname)
	kv.SelectDatabase(1)
	if err := kv.Set(testkey, testvalue0); err != nil {
		t.Errorf("Error, could not set key and value! %s", err.Error())
	}
	if val, err := kv.Get(testkey); err != nil {
		t.Errorf("Error, could not get key! %s", err.Error())
	} else if val != testvalue0 {
		t.Errorf("Error, wrong value! %s != %s", val, testvalue0)
	}
	incval, err := kv.Inc(testkey)
	if err != nil {
		t.Errorf("Error, could not INCR key! %s", err.Error())
	}
	if val, err := kv.Get(testkey); err != nil {
		t.Errorf("Error, could not get key! %s", err.Error())
	} else if val != testvalue1 {
		t.Errorf("Error, wrong value! %s != %s", val, testvalue1)
	} else if incval != testvalue1 {
		t.Errorf("Error, wrong inc value! %s != %s", incval, testvalue1)
	}
	kv.Remove()
	if _, err := kv.Get(testkey); err == nil {
		t.Errorf("Error, could get key! %s", err.Error())
	}
	// Creates "0" and increases the value with 1
	kv.Inc(testkey)
	if val, err := kv.Get(testkey); err != nil {
		t.Errorf("Error, could not get key! %s", err.Error())
	} else if val != testvalue2 {
		t.Errorf("Error, wrong value! %s != %s", val, testvalue2)
	}
	kv.Remove()
	if _, err := kv.Get(testkey); err == nil {
		t.Errorf("Error, could get key! %s", err.Error())
	}
}

func TestTwoFields(t *testing.T) {
	test, test23, ok := twoFields("test1@test2@test3", "@")
	if ok && ((test != "test1") || (test23 != "test2@test3")) {
		t.Error("Error in twoFields functions")
	}
}

func TestICreator(t *testing.T) {
	// Check if the struct comforms to ICreator
	var _ pinterface.ICreator = NewCreator(pool, 1)
}
