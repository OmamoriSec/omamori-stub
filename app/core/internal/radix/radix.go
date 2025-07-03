package radix

import (
	"strings"
)

// Radix Node methods //

type Node struct {
	children  map[string]*Node
	endOfWord bool
	data      string
}

func NewRadixNode() *Node {
	return &Node{
		endOfWord: false,
		children:  nil,
		data:      "",
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

func (node *Node) insert(word string, data string) {
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
	currNode.data = data
}

func (node *Node) delete(word string) bool {
	return node.deleteHelper(word, 0)
}

func (node *Node) deleteHelper(word string, depth int) bool {
	if depth == len(word) {
		// We've reached the end of the word
		if !node.endOfWord {
			return false // Word doesn't exist
		}
		node.endOfWord = false
		node.data = ""

		// If node has no children, it can be deleted
		return !node.hasChildren()
	}

	// Find the child that matches the remaining word
	remainingWord := word[depth:]
	var matchedKey string
	var matchedChild *Node

	for key, child := range node.children {
		if strings.HasPrefix(remainingWord, key) {
			matchedKey = key
			matchedChild = child
			break
		}
	}

	if matchedChild == nil {
		return false // Word doesn't exist
	}

	// Recursively delete from the child
	shouldDeleteChild := matchedChild.deleteHelper(word, depth+len(matchedKey))

	if shouldDeleteChild {
		delete(node.children, matchedKey)

		// If this node has no children after deletion and is not end of word, it can be deleted
		if !node.hasChildren() && !node.endOfWord {
			return true
		}

		// If this node has exactly one child and is not end of word, merge with child
		if len(node.children) == 1 && !node.endOfWord {
			// Get the single child
			for childKey, child := range node.children {
				// Merge: replace this node's children with merged key
				delete(node.children, childKey)

				// Add all grandchildren with merged keys
				for grandchildKey, grandchild := range child.children {
					node.addChild(childKey+grandchildKey, grandchild)
				}

				// If child was end of word, make this node end of word
				if child.endOfWord {
					node.endOfWord = true
					node.data = child.data
				}
				break
			}
		}
	}

	return false
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

func (node *Node) getItems(items *map[string]string, currKey string) {
	// return all the items from the specific node
	if node.endOfWord {
		(*items)[currKey] = node.data
	}

	for edge, child := range node.children {
		child.getItems(items, currKey+edge)
	}
}

// Radix Tree Methods //

type Tree struct {
	root *Node
}

func (tree *Tree) Insert(word string, data string) {
	tree.root.insert(word, data)
}

func (tree *Tree) Search(word string) bool {
	return tree.root.search(word)
}

func (tree *Tree) GetItems() map[string]string {
	items := make(map[string]string)
	tree.root.getItems(&items, "")
	return items

}

func (tree *Tree) Delete(word string) bool {
	return tree.root.delete(word)
}

func NewRadixTree() *Tree {
	return &Tree{root: NewRadixNode()}
}
