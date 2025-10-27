package gobatis

import (
	"bytes"
)

var (
	// SplitPgArrayType split string used by bufio.Scanner Split func
	SplitPgArrayType = SplitByString("{,}")

	// SplitPgArrayStringType split string used by bufio.Scanner Split func
	SplitPgArrayStringType = SplitByString("{,\"}")

	// SplitPgRangeType split string used by bufio.Scanner Split func
	SplitPgRangeType = SplitByString("[,]()")

	// SplitPgRecordType split string used by bufio.Scanner Split func
	SplitPgRecordType = SplitByString(`(,")`)

	// SplitPgArrayRecordType split string used by bufio.Scanner Split func
	SplitPgArrayRecordType = SplitByString(`{}(,")`)

	// SplitPgPointType split string used by bufio.Scanner Split func
	SplitPgPointType = SplitByString(`(,)`)
)

// SplitByString split string used by bufio.Scanner Split func
// any char in chars will be split and return
func SplitByString(chars string) func(data []byte, atEOF bool) (advance int, token []byte, err error) {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		if i := bytes.IndexAny(data, chars); i >= 0 {
			for _, char := range []byte(chars) {
				if data[i] == char {
					if i > 0 {
						return i, dropCR(data[:i]), nil
					} else {
						return i + 1, []byte{data[i]}, nil
					}
				}
			}
		}

		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}
}
