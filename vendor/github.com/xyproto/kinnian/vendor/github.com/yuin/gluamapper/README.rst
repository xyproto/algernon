===============================================================================
gluamapper: maps a GopherLua table to a Go struct
===============================================================================

.. image:: https://godoc.org/github.com/yuin/gluamapper?status.svg
    :target: http://godoc.org/github.com/yuin/gluamapper

.. image:: https://travis-ci.org/yuin/gluamapper.svg
    :target: https://travis-ci.org/yuin/gluamapper

|

gluamapper provides an easy way to map GopherLua tables to Go structs.

gluamapper converts a GopherLua table to ``map[string]interface{}`` , and then converts it to a Go struct using `mapstructure <https://github.com/mitchellh/mapstructure/>`_ .
 
----------------------------------------------------------------
Installation
----------------------------------------------------------------

.. code-block:: bash
   
   go get github.com/yuin/gluamapper

----------------------------------------------------------------
API
----------------------------------------------------------------
See `Go doc <http://godoc.org/github.com/yuin/gluamapper>`_ .

----------------------------------------------------------------
Usage
----------------------------------------------------------------

.. code-block:: go

    type Role struct {
        Name string
    }

    type Person struct {
        Name      string
        Age       int
        WorkPlace string
        Role      []*Role
    }

    L := lua.NewState()
    if err := L.DoString(`
    person = {
      name = "Michel",
      age  = "31", -- weakly input
      work_place = "San Jose",
      role = {
        {
          name = "Administrator"
        },
        {
          name = "Operator"
        }
      }
    }
    `); err != nil {
        panic(err)
    }
    var person Person
    if err := gluamapper.Map(L.GetGlobal("person").(*lua.LTable), &person); err != nil {
        panic(err)
    }
    fmt.Printf("%s %d", person.Name, person.Age)

----------------------------------------------------------------
License
----------------------------------------------------------------
MIT

----------------------------------------------------------------
Author
----------------------------------------------------------------
Yusuke Inuzuka
