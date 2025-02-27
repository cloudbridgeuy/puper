package display

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type DisplayBuilder struct {
	inner *display
}

func NewDisplayBuilder() *DisplayBuilder {
	return &DisplayBuilder{
		inner: &display{writer: io.Writer(os.Stdout)},
	}
}

func (b *DisplayBuilder) WithWriter(w io.Writer) *DisplayBuilder {
	b.inner.writer = w
	return b
}

func (b *DisplayBuilder) WithAttributes(value bool) *DisplayBuilder {
	b.inner.attributes = value
	return b
}

func (b *DisplayBuilder) WithSpan(value bool) *DisplayBuilder {
	b.inner.span = value
	return b
}

func (b *DisplayBuilder) Build() *display {
	return b.inner
}

type display struct {
	attributes bool
	span       bool
	writer     io.Writer
}

func (d display) Print(nodes []*html.Node) {
	for _, node := range nodes {
		d.PrintNode(node, 0)
	}
}

// PrintNode prints the node and its children.
func (d display) PrintNode(n *html.Node, level int) {
	switch n.Type {
	case html.TextNode:
		s := n.Data
		s = strings.TrimSpace(s)
		if s != "" {
			d.PrintIndent(level)
			fmt.Fprintf(d.writer, "%s\n", s)
		}
	case html.ElementNode:
		d.PrintIndent(level)
		if n.DataAtom == atom.Pre {
			d.PrintPre(n)
			return
		}
		if n.DataAtom == atom.Span && !d.span {
			d.PrintChildren(n, level)
			return
		}
		fmt.Fprintf(d.writer, "<%s", n.Data)
		for _, a := range n.Attr {
			if !d.attributes && a.Key != "href" && a.Key != "id" {
				continue
			}
			val := a.Val
			fmt.Fprintf(d.writer, ` %s="%s"`, a.Key, val)
		}
		fmt.Fprintf(d.writer, ">\n")

		if !IsVoidElement(n) {
			d.PrintChildren(n, level+1)
			d.PrintIndent(level)
			fmt.Fprintf(d.writer, "</%s>\n", n.Data)
		}
	case html.DoctypeNode, html.DocumentNode:
		d.PrintChildren(n, level)
	}
}

// PrintChildren prints the children of the node.
func (d display) PrintChildren(n *html.Node, level int) {
	child := n.FirstChild
	for child != nil {
		d.PrintNode(child, level)
		child = child.NextSibling
	}
}

func (d display) PrintIndent(level int) {
	for ; level > 0; level-- {
		fmt.Fprintf(d.writer, " ")
	}
}

// PrintPre prints `<pre></pre>` tags as they come.
func (d display) PrintPre(n *html.Node) {
	switch n.Type {
	case html.TextNode:
		s := n.Data
		fmt.Fprintf(d.writer, "%s", s)
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			d.PrintPre(c)
		}
	case html.ElementNode:
		if n.DataAtom == atom.Span && !d.span {
			if !IsVoidElement(n) {
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					d.PrintPre(c)
				}
			}
			return
		}
		fmt.Fprintf(d.writer, "<%s", n.Data)
		if d.attributes {
			for _, a := range n.Attr {
				if !d.attributes && a.Key != "href" && a.Key != "id" {
					continue
				}
				val := a.Val
				fmt.Fprintf(d.writer, ` %s="%s"`, a.Key, val)
			}
		}
		fmt.Fprintf(d.writer, ">")
		if !IsVoidElement(n) {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				d.PrintPre(c)
			}
			if n.DataAtom == atom.Pre {
				fmt.Fprintf(d.writer, "</%s>\n", n.Data)
			} else {
				fmt.Fprintf(d.writer, "</%s>", n.Data)
			}
		}
	case html.DoctypeNode, html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			d.PrintPre(c)
		}
	}
}

// IsVoidElement returns true if the node is a void element.
func IsVoidElement(n *html.Node) bool {
	switch n.DataAtom {
	case atom.Area, atom.Base, atom.Br, atom.Col, atom.Command, atom.Embed,
		atom.Hr, atom.Img, atom.Input, atom.Keygen, atom.Link,
		atom.Meta, atom.Param, atom.Source, atom.Track, atom.Wbr:
		return true
	}
	return false
}
