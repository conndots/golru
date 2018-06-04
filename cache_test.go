package golru_test

import (
	"testing"
	"time"
	"github.com/conndots/golru"
	"fmt"
)

func TestCache(t *testing.T) {
	conf := golru.NewConfig()
	conf.OnEvict = func(k string, v interface{}) {
		fmt.Println("Evict ", k, v)
	}
	c := golru.New(5, conf)

	c.SetNX("toExpire", "hahaha", 1*time.Second)
	item, exist := c.Get("toExpire")
	if !exist || item == nil || item.Value != "hahaha" {
		t.Errorf("SetNX error: %v", item)
	}

	time.Sleep(time.Second)
	item, exist = c.Get("toExpire")
	if exist || item != nil {
		t.Errorf("SetNX expire error")
	}

	for i := 0; i < 5; i++ {
		c.Set(fmt.Sprint(i), fmt.Sprint(i))
	}
	c.Set("toEvict", "haha")
	time.Sleep(500 * time.Millisecond)
	i, e := c.Get("0")
	if e || i != nil {
		t.Errorf("maxsize error")
	}

	c.ManualGC()
}
