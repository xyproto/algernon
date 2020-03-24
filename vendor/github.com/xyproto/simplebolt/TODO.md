# Plans

- [ ] Add a ForEach function for each of the datatypes, for this one, for db and for simpleredis.
- [ ] Improve the hash map implementation.

## On linkedlist data structure

- [ ] `func(ll *LinkedList) MoveAfter(it *Item, mark *Item) error`
- [ ] `func(ll *LinkedList) MoveBefore(it *Item, mark *Item) error`
- [ ] `InsertAfter`, `InsertBefore`, `MoveAfter` and `MoveBefore`'s mark parameter should be an `Item` - returned from `Front()`, `Back()` or any of the `Getters`
