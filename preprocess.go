package gobatis

import (
	"bufio"
	"bytes"
	"strings"
)

func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

func SplitForXmlAttr(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := bytes.IndexAny(data, `<>"`); i >= 0 {
		switch data[i] {
		case '<', '>', '"':
			if i > 0 {
				return i, dropCR(data[:i]), nil
			} else {
				return i + 1, []byte{data[i]}, nil
			}
		}
		// We have a full newline-terminated line.
		return i + 1, dropCR(data[:i]), nil
	}

	if atEOF {
		return len(data), dropCR(data), nil
	}
	return 0, nil, nil
}

// preprocessXMLReplace preprocess xml
// replace
// & -> &amp;
// > -> &gt;
// < -> &lt;
// ' -> &apos;
// " -> &quot;
func preprocessXMLReplace(mapper []byte) ([]byte, error) {
	scan := bufio.NewScanner(bytes.NewReader(mapper))
	scan.Split(SplitForXmlAttr)

	var (
		inTag   bool
		inQuote bool

		buffer = bytes.NewBuffer(nil)
	)
	for scan.Scan() {
		text := scan.Text()
		switch {
		case text == `<` && !inTag && !inQuote:
			buffer.WriteString(text)
			inTag = true
		case text == `>` && inTag && !inQuote:
			buffer.WriteString(text)
			inTag = false
		case text == `"`:
			buffer.WriteString(text)
			inQuote = !inQuote
		default:
			if inTag && inQuote {
				text = strings.NewReplacer(
					`&amp;`, `&`,
					`&lt;`, `<`,
					`&gt;`, `>`,
					`&apos`, `'`,
					`&quot;`, `"`,
				).Replace(text)
				text = strings.ReplaceAll(text, `&`, `&amp;`)
				text = strings.ReplaceAll(text, `>`, `&gt;`)
				text = strings.ReplaceAll(text, `<`, `&lt;`)
				text = strings.ReplaceAll(text, `'`, `&apos;`)
				text = strings.ReplaceAll(text, `"`, `&quot;`)
				buffer.WriteString(text)
			} else {
				buffer.WriteString(text)
			}
		}
	}

	return buffer.Bytes(), nil
}
