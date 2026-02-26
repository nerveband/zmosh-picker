package picker

const (
	// numbersFirst is the default key sequence: digits 1-9, then letters (skipping 'c' and 'k').
	numbersFirst = "123456789abdefghijlmnopqrstuvwxy"
	// lettersFirst puts letters before digits (still skipping 'c' and 'k').
	lettersFirst = "abdefghijlmnopqrstuvwxy123456789"
)

// keyChars maps session indices to keypress characters.
// Note: 'c' is reserved for custom name, 'k' is reserved for kill mode.
var keyChars = []byte(numbersFirst)

// MaxSessions is the maximum number of sessions the picker can display.
var MaxSessions = len(keyChars)

// LoadKeyMode sets the key character sequence based on the given mode.
// "letters" puts letters first; any other value (including "numbers") uses the default digits-first order.
func LoadKeyMode(mode string) {
	if mode == "letters" {
		keyChars = []byte(lettersFirst)
	} else {
		keyChars = []byte(numbersFirst)
	}
	MaxSessions = len(keyChars)
}

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
