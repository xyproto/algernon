package simplebolt

import (
	"github.com/xyproto/pinterface"
	"os"
	"path"
	"testing"
)

func TestList(t *testing.T) {
	const (
		listname = "abc123_test_test_test_123abc"
		testdata = "123abc"
	)
	db, err := New(path.Join(os.TempDir(), "bolt.db"))
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	list, err := NewList(db, listname)
	if err != nil {
		t.Error(err)
	}
	if err := list.Add(testdata); err != nil {
		t.Errorf("Error, could not add item to list! %s", err.Error())
	}
	items, err := list.GetAll()
	if len(items) != 1 {
		t.Errorf("Error, wrong list length! %v", len(items))
	}
	if (len(items) > 0) && (items[0] != testdata) {
		t.Errorf("Error, wrong list contents! %v", items)
	}
	err = list.Clear()
	if err != nil {
		t.Errorf("Error, could not clear list! %s", err.Error())
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
	db, err := New(path.Join(os.TempDir(), "bolt.db"))
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	kv, err := NewKeyValue(db, kvname)
	if err != nil {
		t.Error(err)
	}
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
	db, err := New(path.Join(os.TempDir(), "bolt.db"))
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	kv, err := NewKeyValue(db, kvname)
	if err != nil {
		t.Error(err)
	}
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

func TestVarious(t *testing.T) {
	db, err := New(path.Join(os.TempDir(), "bolt.db"))
	if err != nil {
		t.Error(err)
	}
	defer db.Close()

	kv, err := NewKeyValue(db, "fruit")
	if err != nil {
		t.Error(err)
	}
	if err := kv.Set("banana", "yes"); err != nil {
		t.Error("Could not set a key+value:", err)
	}

	val, err := kv.Get("banana")
	if err != nil {
		t.Error("Could not get value:", err)
	}

	kv.Set("banana", "2")
	kv.Inc("banana")
	_, err = kv.Get("banana")
	if err != nil {
		t.Error(err)
	}

	kv.Inc("fnu")
	_, err = kv.Get("fnu")
	if err != nil {
		t.Error(err)
	}

	val, err = kv.Get("doesnotexist")
	//fmt.Println("does not exist", val, err)

	kv.Remove()

	l, err := NewList(db, "fruit")
	if err != nil {
		t.Error(err)
	}

	l.Add("kiwi")
	l.Add("banana")
	l.Add("pear")
	l.Add("apple")

	if _, err := l.GetAll(); err != nil {
		t.Error(err)
	}

	last, err := l.GetLast()
	if err != nil {
		t.Error(err)
	}
	if last != "apple" {
		t.Error("last one should be apple")
	}

	lastN, err := l.GetLastN(3)
	if err != nil {
		t.Error(err)
	}
	if lastN[0] != "banana" {
		t.Error("banana is wrong")
	}

	l.Remove()

	// Check that the list qualifies for the IList interface
	var _ pinterface.IList = l

	s, err := NewSet(db, "numbers")
	if err != nil {
		t.Error(err)
	}

	s.Add("9")
	s.Add("7")
	s.Add("2")
	s.Add("2")
	s.Add("2")
	s.Add("7")
	s.Add("8")
	_, err = s.GetAll()
	if err != nil {
		t.Error(err)
	}
	s.Clear()
	v, err := s.GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(v) != 0 {
		t.Error("Could not clear set")
	}
	s.Remove()

	// Check that the set qualifies for the ISet interface
	var _ pinterface.ISet = s

	val, err = kv.Inc("counter")
	if (val != "1") || (err != nil) {
		t.Error("counter should be 1 but is", val)
	}
	kv.Remove()

	// Check that the key value qualifies for the IKeyValue interface
	var _ pinterface.IKeyValue = kv

	h, err := NewHashMap(db, "counter")
	if err != nil {
		t.Error(err)
	}

	h.Set("bob", "password", "hunter1")
	h.Set("bob", "email", "bob@zombo.com")

	h.GetAll()

	_, err = h.Has("bob", "password")
	if err != nil {
		t.Error(err)
	}

	_, err = h.Exists("bob")
	if err != nil {
		t.Error(err)
	}

	h.Set("john", "password", "hunter2")
	h.Set("john", "email", "john@zombo.com")

	h.Del("john")
	found, err := h.Exists("john")
	if err != nil {
		t.Error(err)
	}
	if found {
		t.Error("not supposed to exist")
	}

	h.Remove()

	_, err = h.Has("bob", "password")
	if err == nil {
		t.Error("not supposed to exist")
	}

	_, err = h.Exists("bob")
	if err == nil {
		t.Error("not supposed to exist")
	}

	// Check that the hash map qualifies for the IHashMap interface
	var _ pinterface.IHashMap = h
}

func TestInterface(t *testing.T) {

	db, err := New(path.Join(os.TempDir(), "bolt.db"))
	if err != nil {
		t.Error(err)
	}
	defer db.Close()

	// Check that the database qualifies for the IHost interface
	var _ pinterface.IHost = db

	// Check if the struct comforms to ICreator
	var _ pinterface.ICreator = NewCreator(db)
}

func TestHashMap(t *testing.T) {
	const (
		hashname  = "abc123_test_test_test_123abc_123"
		testid    = "bob"
		testidInv = "b:ob"
		testkey   = "password"
		testvalue = "hunter1"
	)
	db, err := New(path.Join(os.TempDir(), "bolt.db"))
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	hash, err := NewHashMap(db, hashname)
	if err != nil {
		t.Error(err)
	}
	if err := hash.Set(testidInv, testkey, testvalue); err == nil {
		t.Error("Should not be allowed to use an element id with \":\"")
	}
	if err := hash.Set(testid, testkey, testvalue); err != nil {
		t.Errorf("Error, could not add item to hash map! %s", err.Error())
	}
	value2, err := hash.Get(testid, testkey)
	if err != nil {
		t.Error(err)
	}
	if value2 != testvalue {
		t.Errorf("Got a different value in return! %s != %s", value2, testvalue)
	}
	items, err := hash.GetAll()
	if len(items) != 1 {
		t.Errorf("Error, wrong hash map length! %v", len(items))
	}
	if (len(items) > 0) && (items[0] != testid) {
		t.Errorf("Error, wrong hash map id! %v", items)
	}
	err = hash.Remove()
	if err != nil {
		t.Errorf("Error, could not remove hash map! %s", err.Error())
	}
}
