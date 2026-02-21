package picker

import "testing"

func TestKeyForIndex(t *testing.T) {
	tests := []struct {
		index int
		key   byte
	}{
		{0, '1'}, {1, '2'}, {8, '9'},
		{9, 'a'}, {10, 'b'}, {11, 'd'}, // skip 'c' (used for custom)
	}
	for _, tt := range tests {
		got := KeyForIndex(tt.index)
		if got != tt.key {
			t.Errorf("index %d: expected '%c', got '%c'", tt.index, tt.key, got)
		}
	}
}

func TestKeyForIndex_OutOfRange(t *testing.T) {
	got := KeyForIndex(-1)
	if got != '?' {
		t.Errorf("expected '?' for -1, got '%c'", got)
	}
	got = KeyForIndex(100)
	if got != '?' {
		t.Errorf("expected '?' for 100, got '%c'", got)
	}
}

func TestIndexForKey(t *testing.T) {
	idx, ok := IndexForKey('1')
	if !ok || idx != 0 {
		t.Errorf("expected 0, got %d (ok=%v)", idx, ok)
	}
	idx, ok = IndexForKey('a')
	if !ok || idx != 9 {
		t.Errorf("expected 9, got %d (ok=%v)", idx, ok)
	}
	_, ok = IndexForKey('c') // reserved
	if ok {
		t.Error("'c' should not be a valid session key")
	}
	_, ok = IndexForKey('k') // reserved
	if ok {
		t.Error("'k' should not be a valid session key")
	}
}

func TestMaxSessions(t *testing.T) {
	if MaxSessions != 32 {
		t.Errorf("expected 32 max sessions, got %d", MaxSessions)
	}
}

func TestKeyCharsNoDuplicates(t *testing.T) {
	seen := make(map[byte]bool)
	for _, k := range keyChars {
		if seen[k] {
			t.Errorf("duplicate key: '%c'", k)
		}
		seen[k] = true
	}
}
