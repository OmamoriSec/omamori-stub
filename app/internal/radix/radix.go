package radix

import (
	"strings"
)

// Radix Node methods //

type RadixNode struct {
	children  map[string]*RadixNode
	endOfWord bool
}

func NewRadixNode() *RadixNode {
	return &RadixNode{
		endOfWord: false,
		children:  nil,
	}
}

func (node *RadixNode) addChild(key string, child *RadixNode) {
	if node.children == nil {
		node.children = make(map[string]*RadixNode)
	}
	node.children[key] = child
}

func (node *RadixNode) hasChildren() bool {
	return node.children != nil && len(node.children) > 0
}

func (node *RadixNode) insert(word string) {
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

func (node *RadixNode) search(word string) bool {
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

func (node *RadixNode) commonPrefixLength(word1 string, word2 string) int {
	i := 0
	for i < min(len(word1), len(word2)) && word1[i] == word2[i] {
		i++
	}
	return i
}

func (node *RadixNode) countNodes() int {
	count := 1
	for _, child := range node.children {
		count += child.countNodes()
	}
	return count
}

// Radix Tree Methods //

type RadixTree struct {
	root *RadixNode
}

func (tree *RadixTree) Insert(word string) {
	tree.root.insert(word)
}

func (tree *RadixTree) Search(word string) bool {
	return tree.root.search(word)
}

func NewRadixTree() *RadixTree {
	return &RadixTree{root: NewRadixNode()}
}
