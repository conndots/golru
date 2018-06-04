package golru

import "container/list"

type Item struct {
	Key        string
	promotions int32
	size       int
	Value      interface{}
	expireTs   int64
	element    *list.Element
}

func newItem(key string, value interface{}, expireTs int64) *Item {
	size := 1
	if s, ok := value.(WithSize); ok {
		size = s.Size()
	}
	return &Item {
		Key:        key,
		promotions: 0,
		size:       size,
		Value:      value,
		expireTs:   expireTs,
	}
}
