package scraper

// Very simple "Set" implementation for storing strings. The implementation
// is not threadsafe.

// stringSet struct
type stringSet struct {
	// Empty structs occupy 0 memory
	list map[string]struct{}
}

// Has returns true if a the stringSet already contains a string otherwise
// returns false.
func (s *stringSet) Has(v string) bool {
	_, ok := s.list[v]
	return ok
}

// Add adds a string to stringSet.
func (s *stringSet) Add(v string) {
	s.list[v] = struct{}{}
}

// Remove removes a string from stringSet.
func (s *stringSet) Remove(v string) {
	delete(s.list, v)
}

// Clear empties the stringSet.
func (s *stringSet) Clear() {
	s.list = make(map[string]struct{})
}

// Size returns the current size.
func (s *stringSet) Size() int {
	return len(s.list)
}

// newStringSet returns a new empty stringSet.
func newStringSet() *stringSet {
	s := &stringSet{}
	s.list = make(map[string]struct{})
	return s
}
