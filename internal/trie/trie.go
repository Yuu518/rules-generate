package trie

import (
	"errors"
	"strings"
)

type node struct {
	leaf     bool
	children map[string]*node
}

func newNode() *node {
	return &node{
		leaf:     false,
		children: make(map[string]*node),
	}
}

type DomainTrie struct {
	root *node
}

func NewDomainTrie() *DomainTrie {
	return &DomainTrie{
		root: newNode(),
	}
}

func (t *DomainTrie) Insert(domain string) (bool, error) {
	if domain == "" {
		return false, errors.New("empty domain")
	}
	parts := strings.Split(domain, ".")

	n := t.root
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]

		if n.leaf {
			return false, nil
		}
		if _, ok := n.children[part]; !ok {
			n.children[part] = newNode()
			if i == 0 {
				n.children[part].leaf = true
				return true, nil
			}
		}
		n = n.children[part]
	}
	return false, nil
}
