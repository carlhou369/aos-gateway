package trie

type node struct {
	child map[rune]*node
	depth int
	end   bool
}

func (n *node) add(word string) {
	wordData := []rune(word)
	insNode := n
	wordLength := len(wordData)
	if wordLength == 0 {
		return
	}
	for i := 0; i < wordLength; i++ {
		d := wordData[i]
		if insNode.child == nil {
			insNode.child = make(map[rune]*node)
			newChild := new(node)
			newChild.depth = insNode.depth + 1
			if i == wordLength-1 {
				newChild.end = true
			}
			insNode.child[d] = newChild
			insNode = newChild
			continue
		}
		if child, ok := insNode.child[d]; ok {
			insNode = child
			continue
		}
		newChild := new(node)
		newChild.depth = insNode.depth + 1
		insNode.child[d] = newChild
		insNode = newChild
	}
	insNode.end = true
}

func (n *node) getAllEdges(content string) []string {
	edgeList := make([]string, 0)
	if content == "" {
		return edgeList
	}
	wordData := []rune(content)
	for {
		if words := n.getEdge(wordData); len(words) != 0 {
			edgeList = append(edgeList, words...)
		}
		if len(wordData) == 0 {
			return edgeList
		}
		wordData = wordData[1:]
	}
}

func (n *node) hasEdge(content string) bool {
	wordData := []rune(content)
	for {
		if n.inEdge(wordData) {
			return true
		}
		if len(wordData) == 0 {
			return false
		}
		wordData = wordData[1:]
	}
}

func (n *node) getEdge(wordData []rune) []string {
	wordMap := make(map[string]bool)
	wordLength := len(wordData)
	wordList := make([]string, 0)
	if wordLength == 0 {
		return wordList
	}
	insNode := n
	for i := 0; i < wordLength; i++ {
		if insNode.child == nil {
			return wordList
		}
		d := wordData[i]
		child, ok := insNode.child[d]
		if !ok {
			return wordList
		}
		insNode = child
		if insNode.end {
			newWord := string(wordData[:i+1])
			if _, ok := wordMap[newWord]; ok {
				continue
			}
			wordList = append(wordList, newWord)
			wordMap[newWord] = true
		}
	}
	return wordList
}

func (n *node) inEdge(wordData []rune) bool {
	wordLength := len(wordData)
	if wordLength == 0 {
		return false
	}
	insNode := n
	for i := 0; i < wordLength; i++ {
		if insNode.child == nil {
			return false
		}
		d := wordData[i]
		child, ok := insNode.child[d]
		if !ok {
			return false
		}
		insNode = child
		if insNode.end {
			return true
		}
	}
	return false
}

func (n *node) equalEdge(word string) bool {
	wordData := []rune(word)
	wordLength := len(wordData)
	if wordLength == 0 {
		return false
	}
	insNode := n
	for i := 0; i < wordLength; i++ {
		if insNode.child == nil {
			return false
		}
		d := wordData[i]
		child, ok := insNode.child[d]
		if !ok {
			return false
		}
		insNode = child
	}
	return insNode.end
}
