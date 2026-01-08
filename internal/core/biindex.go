package core

import "sort"

// BiIndex 保證任意已註冊的 name 與 id（index）之間的對應關係穩定：
// - 刪除某一 id 不會改變其他 id 的對應
// - 支援回收刪除後的 id（freelist），Assign 會先取用回收的 id
// - 支援直接 Set(id, name) 指定某個 id 對應
// 設計為非並發安全（如需並發請由呼叫端加鎖）。

// BiIndex maps between integer ids and string names with stable ids.
// Deleted ids are pushed onto a free list for reuse.
type BiIndex struct {
	idToString map[int]string
	stringToID map[string]int
	freed      []int
	nextID     int
}

// NewBiIndex create a new BiIndex with optional capacity hint.
func NewBiIndex(cap int) *BiIndex {
	if cap < 0 {
		cap = 0
	}
	return &BiIndex{
		idToString: make(map[int]string, cap),
		stringToID: make(map[string]int, cap),
		freed:      make([]int, 0),
		nextID:     0,
	}
}

// Assign assigns a new id for name if it doesn't exist, reusing freed ids if any.
// Returns (id, true) if newly assigned, or (existingID, false) if name existed.
func (s *BiIndex) Assign(name string) (int, bool) {
	if name == "" {
		return -1, false
	}
	if id, ok := s.stringToID[name]; ok {
		return id, false
	}
	var id int
	if len(s.freed) > 0 {
		id = s.freed[len(s.freed)-1]
		s.freed = s.freed[:len(s.freed)-1]
	} else {
		id = s.nextID
		s.nextID++
	}
	s.idToString[id] = name
	s.stringToID[name] = id
	return id, true
}

// Set assigns name to a specific id. Returns previous name at that id ("" if none) and true on success.
// If the name previously existed at a different id, that old mapping is removed.
func (s *BiIndex) Set(id int, name string) (string, bool) {
	if id < 0 || name == "" {
		return "", false
	}
	// remove old mapping for this name if exists
	if oldID, ok := s.stringToID[name]; ok && oldID != id {
		delete(s.idToString, oldID)
	}
	// get previous occupant
	prev := s.idToString[id]
	if prev != "" {
		delete(s.stringToID, prev)
	}
	// set new mapping
	s.idToString[id] = name
	s.stringToID[name] = id
	if id >= s.nextID {
		s.nextID = id + 1
	}
	// ensure id isn't in freed list
	for i := len(s.freed) - 1; i >= 0; i-- {
		if s.freed[i] == id {
			s.freed = append(s.freed[:i], s.freed[i+1:]...)
			break
		}
	}
	return prev, true
}

// Get returns the name for the given id.
func (s *BiIndex) Get(id int) (string, bool) {
	name, ok := s.idToString[id]
	return name, ok
}

// Index returns the id for the given name.
func (s *BiIndex) Index(name string) (int, bool) {
	id, ok := s.stringToID[name]
	return id, ok
}

// DeleteByID removes the mapping at id and returns the deleted name.
// The id is pushed to free list for reuse.
func (s *BiIndex) DeleteByID(id int) (string, bool) {
	if name, ok := s.idToString[id]; ok {
		delete(s.idToString, id)
		delete(s.stringToID, name)
		s.freed = append(s.freed, id)
		return name, true
	}
	return "", false
}

// DeleteAndShift deletes the mapping at id and shifts all mappings with
// ids greater than id down by 1. It returns the deleted name, a mapping
// of oldID->newID for all shifted ids, and true on success. This operation
// is O(n) and will change many ids; mapping helps callers update references.
func (s *BiIndex) DeleteAndShift(id int) (string, map[int]int, bool) {
	// ensure id exists
	if _, ok := s.idToString[id]; !ok {
		return "", nil, false
	}
	deleted := s.idToString[id]

	// collect ids greater than id and sort them
	ids := make([]int, 0)
	for k := range s.idToString {
		if k > id {
			ids = append(ids, k)
		}
	}
	if len(ids) == 0 {
		// simple delete: like DeleteByID but also adjust freed/nextID
		_, ok := s.DeleteByID(id)
		// adjust freed list and nextID
		for i := range s.freed {
			if s.freed[i] > id {
				s.freed[i] = s.freed[i] - 1
			}
		}
		if s.nextID > id {
			s.nextID--
		}
		return deleted, nil, ok
	}

	sort.Ints(ids)
	mapping := make(map[int]int, len(ids))

	// perform shifts in increasing order
	for _, old := range ids {
		name := s.idToString[old]
		newID := old - 1
		s.idToString[newID] = name
		s.stringToID[name] = newID
		mapping[old] = newID
		delete(s.idToString, old)
	}

	// finally remove the requested id mapping
	delete(s.stringToID, deleted)
	// adjust freed list and nextID
	for i := range s.freed {
		if s.freed[i] > id {
			s.freed[i] = s.freed[i] - 1
		}
	}
	if s.nextID > id {
		s.nextID--
	}

	return deleted, mapping, true
}

// DeleteByName removes the mapping for name.
func (s *BiIndex) DeleteByName(name string) bool {
	if id, ok := s.stringToID[name]; ok {
		_, _ = s.DeleteByID(id)
		return true
	}
	return false
}

// Has reports whether the given name exists.
func (s *BiIndex) Has(name string) bool {
	_, ok := s.stringToID[name]
	return ok
}

// Len returns number of active mappings.
func (s *BiIndex) Len() int {
	return len(s.stringToID)
}

// IDs returns a slice of active ids.
func (s *BiIndex) IDs() []int {
	ids := make([]int, 0, len(s.idToString))
	for id := range s.idToString {
		ids = append(ids, id)
	}
	return ids
}

// Clone creates a deep copy of the BiIndex including reuse metadata.
func (s *BiIndex) Clone() *BiIndex {
	if s == nil {
		return NewBiIndex(0)
	}
	clone := &BiIndex{
		idToString: make(map[int]string, len(s.idToString)),
		stringToID: make(map[string]int, len(s.stringToID)),
		freed:      make([]int, len(s.freed)),
		nextID:     s.nextID,
	}
	for id, name := range s.idToString {
		clone.idToString[id] = name
	}
	for name, id := range s.stringToID {
		clone.stringToID[name] = id
	}
	copy(clone.freed, s.freed)
	return clone
}

// Clear removes all mappings and resets free list and counters.
func (s *BiIndex) Clear() {
	for k := range s.stringToID {
		delete(s.stringToID, k)
	}
	for k := range s.idToString {
		delete(s.idToString, k)
	}
	s.freed = s.freed[:0]
	s.nextID = 0
}
