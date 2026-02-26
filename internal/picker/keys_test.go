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

func TestLoadKeyMode_Numbers(t *testing.T) {
	LoadKeyMode("numbers")
	if KeyForIndex(0) != '1' {
		t.Errorf("numbers mode: index 0 should be '1', got '%c'", KeyForIndex(0))
	}
	if KeyForIndex(9) != 'a' {
		t.Errorf("numbers mode: index 9 should be 'a', got '%c'", KeyForIndex(9))
	}
}

func TestLoadKeyMode_Letters(t *testing.T) {
	LoadKeyMode("letters")
	defer LoadKeyMode("numbers")

	if KeyForIndex(0) != 'a' {
		t.Errorf("letters mode: index 0 should be 'a', got '%c'", KeyForIndex(0))
	}
	if KeyForIndex(2) != 'd' {
		t.Errorf("letters mode: index 2 should be 'd', got '%c'", KeyForIndex(2))
	}
	_, ok := IndexForKey('1')
	if !ok {
		t.Error("letters mode: '1' should still be a valid key")
	}
}

func TestLoadKeyMode_LettersSkipsReserved(t *testing.T) {
	LoadKeyMode("letters")
	defer LoadKeyMode("numbers")

	_, ok := IndexForKey('c')
	if ok {
		t.Error("'c' should still be reserved in letters mode")
	}
	_, ok = IndexForKey('k')
	if ok {
		t.Error("'k' should still be reserved in letters mode")
	}
}

func TestLoadKeyMode_LettersMaxSessions(t *testing.T) {
	LoadKeyMode("letters")
	defer LoadKeyMode("numbers")

	if MaxSessions != 32 {
		t.Errorf("expected 32 max sessions in letters mode, got %d", MaxSessions)
	}
}
