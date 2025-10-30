package main

import (
	"sort"
	"strings"
)

func (w wallet_entries) sort() {
	sort.Slice(w, func(i, j int) bool {
		return w[i].Height > w[j].Height
	})
}

func (l list) split_to_kv(sep string) (keys []string, values []string) {
	if len(l) == 0 {
		return nil, nil
	}
	for _, each := range l {
		if each == "" {
			continue
		}
		pair := strings.Split(each, sep)
		keys = append(keys, pair[0])
		values = append(values, pair[1])
	}
	return
}
