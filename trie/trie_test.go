package trie

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestNode(t *testing.T) {
	root := new(node)
	wordList := []string{
		"你好", "再见", "123","你",
	}
	for _, word := range wordList {
		root.add(word)
	}
	content := "ahes123as再d见到f你好"
	res := root.getAllEdges(content)
	fmt.Println(res)
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

var root *node
var wordMap map[string]bool

const keyCount = 10000
const keyMaxLength = 20
const msgMaxLength = 1000

func init() {
	rand.Seed(time.Now().UnixNano())
	//init trie
	root = new(node)
	wordMap = make(map[string]bool)
	for i := 0; i < keyCount; i++ {
		n := rand.Intn(keyMaxLength)
		word := RandStringRunes(n)
		root.add(word)
		wordMap[word] = true
	}
}

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func BenchmarkEqual(b *testing.B) {
	b.ResetTimer()
	cnt := 0
	for i := 0; i < b.N; i++ {
		n := rand.Intn(keyMaxLength)
		key := RandStringRunes(n)
		ok := root.equalEdge(key)
		if ok {
			cnt++
		}
	}
}

func BenchmarkHas(b *testing.B) {
	b.ResetTimer()
	cnt := 0
	for i := 0; i < b.N; i++ {
		n := rand.Intn(msgMaxLength)
		msg := RandStringRunes(n)
		ok := root.hasEdge(msg)
		if ok {
			cnt++
		}
	}
}

func BenchmarkMap(b *testing.B) {
	b.ResetTimer()
	cnt := 0
	for i := 0; i < b.N; i++ {
		n := rand.Intn(keyMaxLength)
		key := RandStringRunes(n)
		ok := true
		_, ok = wordMap[key]
		if ok {
			cnt++
		}
	}
}
