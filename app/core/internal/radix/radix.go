package radix

import (
	"strings"
)

// Radix Node methods //

type Node struct {
	children  map[string]*Node
	endOfWord bool
}

func NewRadixNode() *Node {
	return &Node{
		endOfWord: false,
		children:  nil,
	}
}

func (node *Node) addChild(key string, child *Node) {
	if node.children == nil {
		node.children = make(map[string]*Node)
	}
	node.children[key] = child
}

func (node *Node) hasChildren() bool {
	return node.children != nil && len(node.children) > 0
}

func (node *Node) insert(word string) {
	currNode := node

	for len(word) > 0 {
		found := false
		for key, child := range currNode.children {
			commonPrefixLen := currNode.commonPrefixLength(key, word)

			if commonPrefixLen > 0 {
				commonPrefix := key[:commonPrefixLen]
				remainingKey := key[commonPrefixLen:]
				remainingWord := word[commonPrefixLen:]

				if commonPrefixLen < len(key) {
					// split existing key
					newChild := NewRadixNode()
					newChild.addChild(remainingKey, child)
					newChild.endOfWord = false

					// Replace the old key with the new one
					currNode.addChild(commonPrefix, newChild)
					delete(currNode.children, key)
					currNode = newChild

				} else {
					currNode = child
				}
				word = remainingWord
				found = true
				break
			}
		}

		if !found {
			newNode := NewRadixNode()
			currNode.addChild(word, newNode)
			word = ""
			currNode = newNode
		}
	}
	currNode.endOfWord = true
}

func (node *Node) search(word string) bool {
	currNode := node
	for len(word) > 0 {
		found := false
		for key, child := range currNode.children {
			if strings.HasPrefix(word, key) {
				word = word[len(key):]
				currNode = child
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return currNode.endOfWord
}

func (node *Node) commonPrefixLength(word1 string, word2 string) int {
	i := 0
	for i < min(len(word1), len(word2)) && word1[i] == word2[i] {
		i++
	}
	return i
}

func (node *Node) countNodes() int {
	count := 1
	for _, child := range node.children {
		count += child.countNodes()
	}
	return count
}

// Radix Tree Methods //

type Tree struct {
	root *Node
}

func (tree *Tree) Insert(word string) {
	tree.root.insert(word)
}

func (tree *Tree) Search(word string) bool {
	return tree.root.search(word)
}

func NewRadixTree() *Tree {
	return &Tree{root: NewRadixNode()}
}
