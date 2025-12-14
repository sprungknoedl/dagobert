package attck

// implementation "inspired" by https://github.com/elliotchance/orderedmap

import "iter"

// elem is an element of a null terminated (non circular) intrusive doubly
// linked list that contains the key of the correspondent element in the ordered map too.
type elem[K comparable, V any] struct {
	// next and previous pointers in the doubly-linked list of elements.
	// To simplify the implementation, internally a list l is implemented
	// as a ring, such that &l.root is both the next element of the last
	// list element (l.Back()) and the previous element of the first list
	// element (l.Front()).
	next, prev *elem[K, V]

	// The key that corresponds to this element in the ordered map.
	key K

	// The value stored with this element.
	value V
}

// list represents a null terminated (non circular) intrusive doubly linked list.
// The list is immediately usable after instantiation without the need of a dedicated initialization.
type list[K comparable, V any] struct {
	root elem[K, V] // list head and tail
}

// push inserts a new element e with value v at the back of list l and returns e.
func (l *list[K, V]) push(key K, value V) *elem[K, V] {
	e := &elem[K, V]{key: key, value: value}
	if l.root.prev == nil {
		// It's the first element
		l.root.next = e
		l.root.prev = e
		return e
	}

	e.prev = l.root.prev
	l.root.prev.next = e
	l.root.prev = e
	return e
}

type OrderedMap[K comparable, V any] struct {
	kv map[K]*elem[K, V]
	ll list[K, V]
}

func newFromTactics(els []Tactic) *OrderedMap[string, Tactic] {
	om := &OrderedMap[string, Tactic]{kv: make(map[string]*elem[string, Tactic], len(els))}
	for _, el := range els {
		element := om.ll.push(el.ID, el)
		om.kv[el.ID] = element
	}
	return om
}

func newFromTechniques(els []Technique) *OrderedMap[string, Technique] {
	om := &OrderedMap[string, Technique]{kv: make(map[string]*elem[string, Technique], len(els))}
	for _, el := range els {
		element := om.ll.push(el.ID, el)
		om.kv[el.ID] = element
	}
	return om
}

// Has checks if a key exists in the map.
func (m *OrderedMap[K, V]) Has(key K) bool {
	_, exists := m.kv[key]
	return exists
}

// Get returns the value for a key. If the key does not exist, the second return
// parameter will be false and the value will be nil.
func (m *OrderedMap[K, V]) Get(key K) (value V, ok bool) {
	v, ok := m.kv[key]
	if ok {
		value = v.value
	}

	return
}

// Len returns the number of elements in the map.
func (m *OrderedMap[K, V]) Len() int {
	return len(m.kv)
}

// Values returns an iterator that yields all the values in the map starting at
// the front (oldest Set element). To create a slice containing all the map
// values, use the slices.Collect function on the returned iterator.
func (m *OrderedMap[K, V]) Values() iter.Seq[V] {
	return func(yield func(value V) bool) {
		for el := m.ll.root.next; el != nil; el = el.next {
			if !yield(el.value) {
				return
			}
		}
	}
}
