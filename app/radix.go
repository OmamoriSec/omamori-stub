package main

import "strings"

// Radix Node methods //

type RadixNode struct {
	endOfWord bool
	children  map[string]*RadixNode
}

func (node *RadixNode) insert(word string) {
	currNode := node
	found := false

	for len(word) > 0 {
		for key, child := range currNode.children {
			commonPrefixLen := currNode.commonPrefixLength(key, word)

			if commonPrefixLen > 0 {
				commonPrefix := key[:commonPrefixLen]
				remainingKey := key[commonPrefixLen:]
				remainingWord := word[commonPrefixLen:]

				if commonPrefixLen < len(key) {
					// split existing key
					newChild := NewRadixNode()
					newChild.children[remainingKey] = child
					newChild.endOfWord = false

					// Replace the old key with the new one
					currNode.children[commonPrefix] = newChild
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
			currNode.children[word] = NewRadixNode()
			currNode.children[word].endOfWord = true
			return
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

func NewRadixNode() *RadixNode {
	return &RadixNode{
		endOfWord: false,
		children:  make(map[string]*RadixNode),
	}
}

// Radix Tree Methods //

type RadixTree struct {
	root *RadixNode
}

func (tree *RadixTree) insert(word string) {
	tree.root.insert(word)
}

func (tree *RadixTree) search(word string) bool {
	return tree.root.search(word)
}
