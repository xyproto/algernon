// Copyright (c) 2026 The liustack/sqlite Authors.
// SPDX-License-Identifier: MIT
// Use of this source code is governed by the MIT license in LICENSE at the
// root of this repository.

//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && (amd64 || arm64))

package csqlite

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// malloc, realloc and free of the C. SQLite's memory is mmap memory, which
// the Go garbage collector does not see: the C keeps pointers to it in it.
//
// A block of up to 256 KiB is taken from a size class. Each class has spans:
// a span is a mapping of its own, of about 256 KiB, cut into blocks of the
// class, with its own list of free blocks threaded through them and a count
// of the blocks in use. A span whose blocks are all free again is unmapped,
// but for a few per class (keepEmpty), once the class has not needed it for
// spanDecay: memory taken at a peak goes back to the system when the peak
// is over, and blocks taken and given back over and over do not map and
// unmap a span each time, as the pages of a reader's cache are: it gives
// them all back when another connection has committed, and takes them
// again as it reads. A use of a class block after it is freed, a bug in the
// C, can fault the process when its span is gone, as it can for a larger
// block. A
// larger block is a mapping of its own, given back when freed. The classes
// go past 64 KiB because SQLite keeps a statement journal in memory in
// chunks of 64 KiB and a little more, one for each SAVEPOINT that writes:
// as mappings, that was an mmap and a munmap per statement. glibc also
// serves blocks of up to 128 KiB or more from its heap. Each block starts
// with a header of 16 bytes (the C asks for 8-byte alignment, and gets 16).
//
// The classes come in arenas, each with its own spans and locks. A
// connection takes its blocks from the arena its stack got when it was made
// (takeArena), so connections that run at the same time do not wait for each
// other at every malloc and free: a reader takes and gives back a block for
// each page it reads after another connection has committed. A block goes
// back to the span it came from, whoever frees it. An arena that no stack
// has used for a while gives the empty spans it kept back (dropArena).
const (
	header     = 16
	largeLimit = 256 << 10 // the largest class, header included
	spanTarget = 256 << 10 // about the size of a span
	spanHeader = 64        // the span struct at the start of a span
	magic      = 0x51_4c_69_74
	// arenas is a power of two: arenaOf masks with it.
	arenas = 32
)

var classSizes = [...]uint32{
	32, 48, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384, 448, 512, 640, 768, 896, 1024,
	1280, 1536, 1792, 2048, 2560, 3072, 3584, 4096, 4352, 5120, 6144, 7168, 8192, 8448,
	10240, 12288, 14336, 16384, 20480, 24576, 28672, 32768, 40960, 49152, 57344, 65536,
	69632, 81920, 98304, 114688, 131072, 163840, 196608, 229376, 262144,
}

// classOf[i] is the class of a block of up to 16*i bytes, header included.
var classOf [largeLimit/16 + 1]uint8

// spanBytes[c] is the size of a span of class c: as many blocks as fit in
// about spanTarget, at least one, rounded up to whole pages.
var spanBytes [len(classSizes)]uintptr

// keepEmpty[c] is how many spans of class c an arena that is in use keeps
// with no block in use, however long they stay so: enough for 4 blocks.
// Only the classes past 64 KiB keep more than one.
var keepEmpty [len(classSizes)]int

// spanDecay is how long a class goes without needing an empty span past
// keepEmpty before a span is unmapped, in nanoseconds. A class looks once
// each spanDecay while it has such spans, and unmaps as many as were empty
// all the time since its last look, whichever of its empty spans they are:
// what a class no longer needs goes back within two spanDecay. The tests
// set it.
var spanDecay atomic.Int64

func init() { spanDecay.Store(int64(time.Second)) }

func init() {
	c := 0
	for i := range classOf {
		for uint32(16*i) > classSizes[c] {
			c++
		}
		classOf[i] = uint8(c)
	}
	for c, size := range classSizes {
		k := max((spanTarget-spanHeader)/uintptr(size), 1)
		spanBytes[c] = roundUp(spanHeader+k*uintptr(size), pageSize)
		keepEmpty[c] = int((4 + k - 1) / k)
	}
}

// A span is at the start of its mapping. Its blocks follow the struct: the
// free ones are on free, and the ones never taken start at fresh.
type span struct {
	free       uintptr // the first free block (its header), 0 for none
	fresh, end uintptr // the blocks never taken: fresh up to end
	used       uintptr // the blocks in use
	prev, next uintptr // in the class's list of spans with a free block
	listed     bool
	arena      uint8 // the arena of the span, which never changes
}

// The span struct fits in spanHeader, which keeps the blocks 16-byte aligned.
var _ [spanHeader - unsafe.Sizeof(span{})]byte

func spanAt(s uintptr) *span { return (*span)(at(s)) }

// hasFree says whether the span can give a block.
func (s *span) hasFree(size uintptr) bool { return s.free != 0 || s.fresh+size <= s.end }

type sizeClass struct {
	mu    sync.Mutex
	spans uintptr // the first span with a free block
	empty int     // spans of the class with no block in use
	// low is the fewest empty spans the class had since its last look, or
	// since its timer was set if no look came since, never more than
	// empty. timer is the next look: set whenever empty is more than
	// keepEmpty.
	low   int
	timer *time.Timer
	_     [24]byte
}

// A sizeClass is a cache line.
var _ [0]byte = [unsafe.Sizeof(sizeClass{}) - 64]byte{}

type arena struct {
	classes [len(classSizes)]sizeClass
}

var heap struct {
	arenas [arenas]arena
}

// arenaUse counts the stacks that take from each arena.
var arenaUse struct {
	mu sync.Mutex
	n  [arenas]int
	// left counts the times each arena was left with no stack. timer, when
	// set, will run trimIdle for the arena, and seen is what left was when
	// the timer was set.
	left, seen [arenas]uint64
	timer      [arenas]*time.Timer
}

// idleTrim is how long an arena has no stack before it gives its empty spans
// back. It is read and, by the tests, set with arenaUse.mu held.
var idleTrim = time.Second

// takeArena gives a new stack the arena the fewest stacks take from, the
// first of them: a program with one connection uses one arena, and one
// with up to as many connections as arenas has an arena for each.
func takeArena() uint8 {
	arenaUse.mu.Lock()
	a := 0
	for i, n := range arenaUse.n {
		if n < arenaUse.n[a] {
			a = i
		}
	}
	arenaUse.n[a]++
	arenaUse.mu.Unlock()
	return uint8(a)
}

// dropArena is called for a stack that is freed. An arena left with no stack
// still holds the empty spans its classes keep. A program that had many
// connections for a moment would hold those of every arena for good, so
// they go back once the arena has had no stack for idleTrim. They do not go
// back at once: database/sql closes and opens connections all the time when
// more are in use than it may keep idle, and each new one would map the
// spans again.
func dropArena(a uint8) {
	arenaUse.mu.Lock()
	arenaUse.n[a]--
	if arenaUse.n[a] == 0 {
		arenaUse.left[a]++
		if arenaUse.timer[a] == nil {
			arenaUse.seen[a] = arenaUse.left[a]
			arenaUse.timer[a] = time.AfterFunc(idleTrim, func() { trimIdle(a) })
		}
	}
	arenaUse.mu.Unlock()
}

// trimIdle is run by the timer of arena a, idleTrim after the arena was left
// with no stack. It gives the arena's empty spans back if no stack took it
// since, and waits once more if one took it and left it again.
func trimIdle(a uint8) {
	arenaUse.mu.Lock()
	switch {
	case arenaUse.n[a] != 0:
		// In use. The next time it is left sets a timer.
		arenaUse.timer[a] = nil
	case arenaUse.left[a] != arenaUse.seen[a]:
		// Taken and left again since, so not idle for that long yet.
		arenaUse.seen[a] = arenaUse.left[a]
		arenaUse.timer[a] = time.AfterFunc(idleTrim, func() { trimIdle(a) })
	default:
		arenaUse.timer[a] = nil
		arenaUse.mu.Unlock()
		heap.arenas[a].trim()
		return
	}
	arenaUse.mu.Unlock()
}

// trim unmaps the spans of the arena that have no block in use. A stack
// that takes the arena meanwhile maps what it needs, as in a new arena.
func (a *arena) trim() {
	for ci := range a.classes {
		c := &a.classes[ci]
		c.mu.Lock()
		gone := c.takeEmpty(c.empty)
		c.mu.Unlock()
		unmapSpans(gone, ci)
	}
}

// takeEmpty takes n spans with no block in use out of the class, with c.mu
// held, and returns them linked by next for unmapSpans.
func (c *sizeClass) takeEmpty(n int) (gone uintptr) {
	for sb := c.spans; sb != 0 && n > 0; {
		s := spanAt(sb)
		next := s.next
		if s.used == 0 {
			c.unlink(sb)
			c.empty--
			n--
			s.next = gone
			gone = sb
		}
		sb = next
	}
	c.low = min(c.low, c.empty)
	return gone
}

// unmapSpans unmaps the spans of class ci that takeEmpty gave.
func unmapSpans(gone uintptr, ci int) {
	for gone != 0 {
		next := spanAt(gone).next
		munmap(gone, spanBytes[ci])
		mappedSpans.Add(-int64(spanBytes[ci]))
		gone = next
	}
}

// look is run by the timer of the class, of class ci. It unmaps the spans
// past keepEmpty that were empty all the time since the last look, and
// looks again later while the class has more empty spans than it keeps.
func (c *sizeClass) look(ci int) {
	c.mu.Lock()
	gone := c.decay(ci)
	c.timer = nil
	if c.empty > keepEmpty[ci] {
		c.setTimer(ci)
	}
	c.mu.Unlock()
	unmapSpans(gone, ci)
}

// decay is a look without its timer, with c.mu held. It takes the spans
// that were empty all the time since the last look, and leaves keepEmpty
// empty ones, the spans the class took blocks from since among them.
func (c *sizeClass) decay(ci int) (gone uintptr) {
	gone = c.takeEmpty(min(c.low, c.empty-keepEmpty[ci]))
	c.low = c.empty
	return gone
}

// setTimer sets the next look of the class, with c.mu held.
func (c *sizeClass) setTimer(ci int) {
	c.timer = time.AfterFunc(time.Duration(spanDecay.Load()), func() { c.look(ci) })
}

// arenaOf is the arena of the stack sp is in. The mask keeps a header the C
// damaged from reaching past the arenas.
func arenaOf(sp uintptr) *arena {
	return &heap.arenas[*(*uint8)(at(stackBase(sp) + arenaOffset))&(arenas-1)]
}

// mappedSpans is the bytes of the spans mapped now, and spanMaps the spans
// mapped so far.
var (
	mappedSpans atomic.Int64
	spanMaps    atomic.Int64
)

// blockHeader is at the start of each block: its class plus one (0 for a
// mapping of its own), a check word, and the length of a mapping or the
// span of a class block.
type blockHeader struct {
	class uint32
	check uint32
	size  uint64
}

func headerOf(p uintptr) *blockHeader { return (*blockHeader)(at(p - header)) }

// A bad block (heap damage) fails the call: free leaks it, realloc says out of
// memory, and the connection is marked broken. The heap is shared, but the
// block that failed is the only one lost.
func malloc(sp, n uintptr) uintptr { return arenaOf(sp).alloc(n) }

func free(sp, p uintptr) {
	defer failQuiet(sp)
	release(p)
}

func realloc(sp, p, n uintptr) (ret uintptr) {
	defer failWith(sp, &ret, 0)
	a := arenaOf(sp)
	if p == 0 {
		return a.alloc(n)
	}
	h := headerOf(p)
	if h.check != magic {
		panic("csqlite: realloc of a block malloc did not give")
	}
	var have uintptr
	if h.class == 0 {
		have = uintptr(h.size) - header
	} else {
		have = uintptr(classSizes[h.class-1]) - header
	}
	if n <= have && (h.class == 0 || n+header > have/2) {
		return p
	}
	q := a.alloc(n)
	if q == 0 {
		return 0
	}
	copy(bytesAt(q, int(min(n, have))), bytesAt(p, int(min(n, have))))
	release(p)
	return q
}

// allocFault is the function SetAllocFault set, nil when none is.
var allocFault atomic.Pointer[func(n uintptr) bool]

// SetAllocFault has every malloc and realloc of the C ask f first, with the
// bytes asked for: when f says true the C gets no memory, as when the system
// has none to give. It is for tests that check what SQLite and the driver do
// when memory runs out. f is called from every goroutine that allocates. nil
// turns it off.
func SetAllocFault(f func(n uintptr) bool) {
	if f == nil {
		allocFault.Store(nil)
		return
	}
	allocFault.Store(&f)
}

// alloc is malloc for the Go side, which has no stack at hand: it takes
// from the first arena.
func alloc(n uintptr) uintptr { return heap.arenas[0].alloc(n) }

func (a *arena) alloc(n uintptr) uintptr {
	if n > 1<<40 {
		return 0
	}
	if f := allocFault.Load(); f != nil && (*f)(n) {
		return 0
	}
	t := n + header
	if t > largeLimit {
		return allocLarge(t)
	}
	ci := classOf[(t+15)/16]
	size := uintptr(classSizes[ci])
	c := &a.classes[ci]
	c.mu.Lock()
	sb := c.spans
	if sb == 0 {
		if sb = a.newSpan(ci); sb == 0 {
			c.mu.Unlock()
			return 0
		}
		c.push(sb)
		c.empty++
	}
	s := spanAt(sb)
	b := s.free
	if b != 0 {
		s.free = *(*uintptr)(at(b + header))
	} else {
		b = s.fresh
		s.fresh += size
	}
	if s.used == 0 {
		c.empty--
		c.low = min(c.low, c.empty)
	}
	s.used++
	if !s.hasFree(size) {
		c.unlink(sb)
	}
	c.mu.Unlock()
	h := (*blockHeader)(at(b))
	h.class = uint32(ci) + 1
	h.check = magic
	h.size = uint64(sb)
	return b + header
}

// newSpan maps a span of class ci, with all its blocks fresh.
func (a *arena) newSpan(ci uint8) uintptr {
	m, err := mmap(spanBytes[ci])
	if err != nil {
		return 0
	}
	mappedSpans.Add(int64(spanBytes[ci]))
	spanMaps.Add(1)
	sb := uintptr(m)
	*spanAt(sb) = span{
		fresh: sb + spanHeader,
		end:   sb + spanBytes[ci],
		arena: uint8((uintptr(unsafe.Pointer(a)) - uintptr(unsafe.Pointer(&heap.arenas[0]))) / unsafe.Sizeof(*a)),
	}
	return sb
}

// push puts the span sb first in the class's list, with c.mu held.
func (c *sizeClass) push(sb uintptr) {
	s := spanAt(sb)
	s.prev, s.next, s.listed = 0, c.spans, true
	if c.spans != 0 {
		spanAt(c.spans).prev = sb
	}
	c.spans = sb
}

// unlink takes the span sb out of the class's list, with c.mu held.
func (c *sizeClass) unlink(sb uintptr) {
	s := spanAt(sb)
	if s.prev != 0 {
		spanAt(s.prev).next = s.next
	} else {
		c.spans = s.next
	}
	if s.next != 0 {
		spanAt(s.next).prev = s.prev
	}
	s.prev, s.next, s.listed = 0, 0, false
}

func allocLarge(t uintptr) uintptr {
	t = roundUp(t, pageSize)
	m, err := mmap(t)
	if err != nil {
		return 0
	}
	b := uintptr(m)
	h := (*blockHeader)(at(b))
	h.class = 0
	h.check = magic
	h.size = uint64(t)
	return b + header
}

func release(p uintptr) {
	if p == 0 {
		return
	}
	h := headerOf(p)
	if h.check != magic {
		panic("csqlite: free of a block malloc did not give")
	}
	b := p - header
	if h.class == 0 {
		munmap(b, uintptr(h.size))
		return
	}
	ci := h.class - 1
	sb := uintptr(h.size)
	s := spanAt(sb)
	c := &heap.arenas[s.arena&(arenas-1)].classes[ci]
	h.check = 0
	c.mu.Lock()
	*(*uintptr)(at(p)) = s.free
	s.free = b
	s.used--
	if !s.listed {
		c.push(sb)
	}
	if s.used > 0 {
		c.mu.Unlock()
		return
	}
	// The span stays for the next blocks of the class. Past keepEmpty, a
	// look unmaps it if the class does not need it.
	c.empty++
	if c.empty > keepEmpty[ci] && c.timer == nil {
		c.low = c.empty
		c.setTimer(int(ci))
	}
	c.mu.Unlock()
}
