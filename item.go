package golru

import "container/list"

type Item struct {
	Key        string
	promotions int32
	weight     int
	Value      interface{}
	expireTs   int64
	element    *list.Element
}

func newItem(key string, value interface{}, expireTs int64) *Item {
	weight := 1
	if s, ok := value.(WithWeight); ok {
		weight = s.Weight()
	}
	return &Item {
		Key:        key,
		promotions: 0,
		weight:     weight,
		Value:      value,
		expireTs:   expireTs,
	}
}
