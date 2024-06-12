package trie

import (
	"encoding/csv"
	"os"
	"strings"
)

type strmap map[string]bool

var SensitiveRoot *node
var SensitiveMap map[string]strmap
var SingleSensitive map[string]bool

func LoadSensitive(filePath string) error {
	SensitiveRoot = new(node)
	SensitiveMap = make(map[string]strmap)
	SingleSensitive = make(map[string]bool)

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		return err
	}

	for _, record := range records {
		if len(record) == 0 {
			continue
		}
		sensitive := record[0]
		if sensitive == "" {
			continue
		}
		words := strings.SplitN(sensitive, " ", 2)
		for _, word := range words {
			if word == "" {
				continue
			}
			SensitiveRoot.add(word)
		}
		if len(words) == 1 {
			SingleSensitive[words[0]] = true
		}
		if len(words) == 2 {
			if m, ok := SensitiveMap[words[0]]; ok {
				m[words[1]] = true
			} else {
				SensitiveMap[words[0]] = map[string]bool{words[1]: true}
			}
		}
	}
	return nil
}

func IsSensitive(content string) bool {
	words := SensitiveRoot.getAllEdges(content)
	if len(words) == 0 {
		return false
	}
	for _, word := range words {
		if _, ok := SingleSensitive[word]; ok {
			return true
		}
		if m, ok := SensitiveMap[word]; ok {
			for _, moreWord := range words {
				if _, ok := m[moreWord]; ok {
					return true
				}
			}
		}
	}
	return false
}
