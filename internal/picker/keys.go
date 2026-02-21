package picker

// keyChars maps session indices to keypress characters.
// Matches the zsh script: 123456789abdefghijlmnopqrstuvwxy
// Note: 'c' is reserved for custom name, 'k' is reserved for kill mode.
var keyChars = []byte("123456789abdefghijlmnopqrstuvwxy")

// MaxSessions is the maximum number of sessions the picker can display.
var MaxSessions = len(keyChars)

// KeyForIndex returns the key character for a session index.
func KeyForIndex(index int) byte {
	if index < 0 || index >= len(keyChars) {
		return '?'
	}
	return keyChars[index]
}

// IndexForKey returns the session index for a key character.
func IndexForKey(key byte) (int, bool) {
	for i, k := range keyChars {
		if k == key {
			return i, true
		}
	}
	return -1, false
}
