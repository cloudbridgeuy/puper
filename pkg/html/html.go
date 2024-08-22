package html

import (
	"fmt"
	"io"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

// ParseHTML parses the HTML while rendering the charset
func ParseHTML(r io.Reader, cs string) (*html.Node, error) {
	var err error

	if cs == "" {
		// Attempt to guess the charset of the HTML document.
		r, err = charset.NewReader(r, "")
		if err != nil {
			return nil, err
		}
	} else {
		// Let the user specify the charset.
		e, name := charset.Lookup(cs)
		if name == "" {
			return nil, fmt.Errorf("unsupported charset: %s", cs)
		}
		r = transform.NewReader(r, e.NewDecoder())
	}
	return html.Parse(r)
}
