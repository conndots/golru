package golru

import (
	"sync"
	"time"
)

type bucket struct {
	sync.RWMutex
	data map[string]*Item
	timer Timer
}

func newBucket(timer Timer) *bucket {
	return &bucket{
		data: make(map[string]*Item, 100),
		timer: timer,
	}
}

func (b *bucket) get(key string) (*Item, bool) {
	b.RLock()
	defer b.RUnlock()
	item, exist := b.data[key]
	return item, exist
}

func (b *bucket) set(key string, value interface{}, expire time.Duration) (*Item, *Item) {
	expireTs := b.timer.NowNano() + expire.Nanoseconds()
	item := newItem(key, value, expireTs)
	b.RLock()
	existedItem := b.data[key]
	b.RUnlock()

	b.Lock()
	defer b.Unlock()
	b.data[key] = item
	return item, existedItem
}

func (b *bucket) remove(key string) *Item {
	b.RLock()
	item := b.data[key]
	b.RUnlock()

	b.Lock()
	defer b.Unlock()
	delete(b.data, key)
	return item
}

//manually remove garbages
func (b *bucket) manualGC() {
	b.RLock()
	prevSize := len(b.data)
	b.RUnlock()

	newData := make(map[string]*Item, prevSize)
	nowStamp := b.timer.NowNano()
	for key, item := range b.data {
		if item.ExpireNano == noExpire || nowStamp < item.ExpireNano {
			newData[key] = item
		}
	}

	b.Lock()
	defer b.Unlock()
	b.data = newData
}