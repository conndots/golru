package golru

import "container/list"

type Item struct {
	Key        string
	promotions int32
	weight     int
	Value      interface{}
	ExpireNano int64
	element    *list.Element
}

func newItem(key string, value interface{}, expireNano int64) *Item {
	weight := 1
	if s, ok := value.(WithWeight); ok {
		weight = s.Weight()
	}
	return &Item {
		Key:        key,
		promotions: 0,
		weight:     weight,
		Value:      value,
		ExpireNano: expireNano,
	}
}
