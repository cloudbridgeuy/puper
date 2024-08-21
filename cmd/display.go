package cmd

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func Display(nodes []*html.Node) {
	for _, node := range nodes {
		PrintNode(node, 0)
	}
}

// PrintNode prints the node and its children.
func PrintNode(n *html.Node, level int) {
	switch n.Type {
	case html.TextNode:
		s := n.Data
		s = strings.TrimSpace(s)
		if s != "" {
			PrintIndent(level)
			fmt.Println(s)
		}
	case html.ElementNode:
		PrintIndent(level)
		fmt.Printf("<%s", n.Data)
		for _, a := range n.Attr {
			val := a.Val
			fmt.Printf(` %s="%s"`, a.Key, val)
		}
		fmt.Println(">")
		if !IsVoidElement(n) {
			PrintChildren(n, level+1)
			PrintIndent(level)
			fmt.Printf("</%s>\n", n.Data)

		}
	case html.CommentNode:
		PrintIndent(level)
		data := n.Data
		fmt.Printf("<!--%s-->\n", data)
		PrintChildren(n, level)
	case html.DoctypeNode, html.DocumentNode:
		PrintChildren(n, level)
	}
}

// PrintChildren prints the children of the node.
func PrintChildren(n *html.Node, level int) {
	child := n.FirstChild
	for child != nil {
		PrintNode(child, level)
		child = child.NextSibling
	}
}

func PrintIndent(level int) {
	for ; level > 0; level-- {
		fmt.Print(" ")
	}
}

// PrintPre prints `<pre></pre>` tags as they come.
func PrintPre(n *html.Node) {
	switch n.Type {
	case html.TextNode:
		s := n.Data
		fmt.Print(s)
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			PrintPre(c)
		}
	case html.ElementNode:
		fmt.Printf("<%s", n.Data)
		for _, a := range n.Attr {
			val := a.Val
			fmt.Printf(` %s="%s"`, a.Key, val)
		}
		fmt.Print(">")
		if !IsVoidElement(n) {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				PrintPre(c)
			}
			fmt.Printf("</%s>", n.Data)
		}
	case html.CommentNode:
		data := n.Data
		fmt.Printf("<!--%s-->\n", data)
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			PrintPre(c)
		}
	case html.DoctypeNode, html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			PrintPre(c)
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
