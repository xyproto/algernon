// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package sqlite

// maxQueries is the most statements a connection keeps for QueryContext.
const maxQueries = 64

// queryStmt gives the statement of QueryContext for query: the one the
// connection keeps for it when its rows are closed, or one prepared now. One
// prepared now is kept when it only reads the main database (read-only, not
// Private, not EXPLAIN) and the cache has room or a statement no rows use to
// let go. Otherwise its rows finalize it.
//
// A write is never kept: SQLite codes some settings of the connection into a
// write as it prepares it, such as foreign_keys and defer_foreign_keys, and
// does not always make it prepare again when they change (a PRAGMA prepared
// and never run, defer_foreign_keys turned off at the end of a transaction).
// Then a kept write would skip a foreign key check. A PRAGMA is not kept
// either, as SQLite prepares one again each time it runs. And conn.prepare
// empties the cache when the connection prepares a statement that changes
// it.
func (c *conn) queryStmt(query string) (*stmt, error) {
	c.queryTick++
	if s := c.queries[query]; s != nil && !s.busy {
		s.used = c.queryTick
		return s, nil
	}
	p, sql, err := c.prepareOne(query)
	if err != nil {
		return nil, err
	}
	s := newStmt(c, p, sql, true)
	if !p.ReadOnly() || p.Private() || p.IsExplain() || c.queries[query] != nil || !c.queryRoom() {
		return s, nil
	}
	if c.queries == nil {
		c.queries = map[string]*stmt{}
	}
	s.once, s.kept, s.used, s.keptAt = false, query, c.queryTick, p.Reprepares()
	c.queries[query] = s
	return s, nil
}

// queryRoom makes room for one more statement in the cache, finalizing the
// one used longest ago of those no rows use. It says false when every one is
// in use.
func (c *conn) queryRoom() bool {
	if len(c.queries) < maxQueries {
		return true
	}
	var old *stmt
	for _, s := range c.queries {
		if !s.busy && (old == nil || s.used < old.used) {
			old = s
		}
	}
	if old == nil {
		return false
	}
	c.dropQuery(old)
	return true
}

// dropQuery takes s out of the cache and finalizes it.
func (c *conn) dropQuery(s *stmt) {
	delete(c.queries, s.kept)
	s.kept = ""
	s.Close()
}

// keepQuery is what the rows of a kept statement do as they close, after the
// reset gave err: the statement goes back to the cache with its arguments
// let go. A statement that failed is finalized instead, so that the next
// query prepares from nothing, and so is one SQLite prepared again after a
// change of the schema: what it reads may have changed, and the next query
// sorts it again (a table may have become a Go table, whose plan is made in
// Go as SQLite prepares the query).
func (c *conn) keepQuery(s *stmt, err error) error {
	if err == nil && s.s.Reprepares() == s.keptAt {
		err = s.s.ClearBindings()
		if err == nil {
			return nil
		}
	}
	c.dropQuery(s)
	return err
}

// dropQueries empties the cache: as the connection closes, or when it
// prepared a statement that changes it. A statement whose rows are still
// open is finalized by its rows.
func (c *conn) dropQueries() {
	for _, s := range c.queries {
		if s.busy {
			delete(c.queries, s.kept)
			s.kept, s.once = "", true
			continue
		}
		c.dropQuery(s)
	}
}
