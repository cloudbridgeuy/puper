package cmd

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type Display struct {
	attributes bool
	span       bool
}

func (d Display) Print(nodes []*html.Node) {
	for _, node := range nodes {
		d.PrintNode(node, 0)
	}
}

// PrintNode prints the node and its children.
func (d Display) PrintNode(n *html.Node, level int) {
	switch n.Type {
	case html.TextNode:
		s := n.Data
		s = strings.TrimSpace(s)
		if s != "" {
			d.PrintIndent(level)
			fmt.Println(s)
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
		fmt.Printf("<%s", n.Data)
		for _, a := range n.Attr {
			if !d.attributes && a.Key != "href" && a.Key != "id" {
				continue
			}
			val := a.Val
			fmt.Printf(` %s="%s"`, a.Key, val)
		}
		fmt.Println(">")

		if !IsVoidElement(n) {
			d.PrintChildren(n, level+1)
			d.PrintIndent(level)
			fmt.Printf("</%s>\n", n.Data)
		}
	case html.CommentNode:
		d.PrintIndent(level)
		data := n.Data
		fmt.Printf("<!--%s-->\n", data)
		d.PrintChildren(n, level)
	case html.DoctypeNode, html.DocumentNode:
		d.PrintChildren(n, level)
	}
}

// PrintChildren prints the children of the node.
func (d Display) PrintChildren(n *html.Node, level int) {
	child := n.FirstChild
	for child != nil {
		d.PrintNode(child, level)
		child = child.NextSibling
	}
}

func (d Display) PrintIndent(level int) {
	for ; level > 0; level-- {
		fmt.Print(" ")
	}
}

// PrintPre prints `<pre></pre>` tags as they come.
func (d Display) PrintPre(n *html.Node) {
	switch n.Type {
	case html.TextNode:
		s := n.Data
		fmt.Print(s)
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
		fmt.Printf("<%s", n.Data)
		if d.attributes {
			for _, a := range n.Attr {
				if !d.attributes && a.Key != "href" && a.Key != "id" {
					continue
				}
				val := a.Val
				fmt.Printf(` %s="%s"`, a.Key, val)
			}
		}
		fmt.Print(">")
		if !IsVoidElement(n) {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				d.PrintPre(c)
			}
			fmt.Printf("</%s>", n.Data)
		}
	case html.CommentNode:
		data := n.Data
		fmt.Printf("<!--%s-->\n", data)
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			d.PrintPre(c)
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
