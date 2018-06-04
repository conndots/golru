package golru

import (
	"container/list"
	"hash/fnv"
	"time"
)

const (
	itemDeleted = -1
	noExpire = -2
)

type Cache struct {
	latestUsage  *list.List
	bucketMask   uint32
	maxWeight    int
	weight       int
	itemSize     int
	buckets      []*bucket
	toRemove     chan *Item
	toPromote    chan *Item
	evictActions chan *Item
	config       *Configuration
}

type WithWeight interface {
	Weight() int
}

type Configuration struct {
	ActionsToPromote int32
	ChannelBuffer    int
	OnEvict          func(string, interface{})
	BucketNum        int
	Timer Timer
}

func NewConfig() *Configuration {
	return &Configuration{
		ActionsToPromote: 1,
		ChannelBuffer: 1024,
		BucketNum: 32,
		OnEvict: nil,
		Timer: &GoTimer{},
	}
}

func New(maxWeight int, config *Configuration) *Cache {
	c := &Cache{
		latestUsage: list.New(),
		bucketMask:  uint32(config.BucketNum - 1),
		buckets:     make([]*bucket, config.BucketNum),
		maxWeight:   maxWeight,
		weight:      0,
		itemSize:    0,
		toRemove:    make(chan *Item, config.ChannelBuffer),
		toPromote:   make(chan *Item, config.ChannelBuffer),
		config:      config,
	}
	if config.OnEvict != nil {
		c.evictActions = make(chan *Item, config.ChannelBuffer)
	}
	for i := 0; i < config.BucketNum; i++ {
		c.buckets[i] = newBucket(config.Timer)
	}

	c.asyncLoop()
	return c
}

func (c *Cache) Size() int {
	return c.itemSize
}
func (c *Cache) TotalWeight() int {
	return c.weight
}
func (c *Cache) promote(item *Item) {
	c.toPromote <- item
}
func (c *Cache) shouldPromote(item *Item) bool {
	if item.promotions == itemDeleted {
		return false
	}
	item.promotions++
	return item.promotions == c.config.ActionsToPromote
}
// return true if new element is added
func (c *Cache) doPromote(item *Item) bool {
	if item.promotions == itemDeleted {
		return false
	}
	if item.element != nil {
		if c.shouldPromote(item) {
			c.latestUsage.MoveToFront(item.element)
			item.promotions = 0
		}
		return false
	}
	c.weight += item.weight
	c.itemSize++
	item.element = c.latestUsage.PushFront(item)
	return true
}
func (c *Cache) remove(item *Item) {
	c.toRemove <- item
}
func (c *Cache) Remove(key string) bool {
	item := c.getBucket(key).remove(key)
	if item != nil {
		c.remove(item)
		return true
	}
	return false
}
func (c *Cache) doRemove(item *Item) {
	if item.element == nil {
		item.promotions = itemDeleted
	} else {
		c.weight -= item.weight
		c.itemSize--
		c.latestUsage.Remove(item.element)
		item.promotions = itemDeleted
	}
}
//manually clean removed data
func (c *Cache) ManualGC() {
	for _, b := range c.buckets {
		b.manualGC()
	}
}
func (c *Cache) clean() {
	elem := c.latestUsage.Back()
	for c.weight > c.maxWeight {
		if elem == nil {
			return
		}
		prev := elem.Prev()
		item := elem.Value.(*Item)

		c.getBucket(item.Key).remove(item.Key)
		c.weight -= item.weight
		c.itemSize--
		c.latestUsage.Remove(elem)
		item.promotions = itemDeleted
		if c.config.OnEvict != nil {
			c.evictActions <- item
		}
		elem = prev
	}
}

func (c *Cache) Set(key string, value interface{}) *Item {
	return c.SetNX(key, value, noExpire)
}

func (c *Cache) SetNX(key string, value interface{}, duration time.Duration) *Item {
	item, existedItem := c.getBucket(key).set(key, value, duration)
	if existedItem != nil {
		c.remove(existedItem)
	}
	c.promote(item)
	return item
}

func (c *Cache) Get(key string) (*Item, bool) {
	bucket := c.getBucket(key)
	item, exist := bucket.get(key)
	if !exist {
		return nil, false
	}

	if item.expireTs == noExpire || item.expireTs > c.config.Timer.NowNano() {
		c.promote(item)
	} else { //已过期
		d := bucket.remove(item.Key)
		if d != nil {
			c.remove(item)
		}
		return nil, false
	}
	return item, true
}

func (c *Cache) getBucket(key string) *bucket {
	h := fnv.New32a()
	h.Write([]byte(key))
	return c.buckets[h.Sum32()&c.bucketMask]
}

func (c *Cache) asyncLoop() {
	go func() {
		for {
			select {
			case item := <-c.toPromote:
				if c.doPromote(item) && c.weight > c.maxWeight { //超过maxSize，触发清理
					c.clean()
				}

			case item := <-c.toRemove:
				c.doRemove(item)
			}
		}
	}()
	if c.config.OnEvict != nil {
		go func() {
			for {
				select {
				case item := <-c.evictActions:
					c.config.OnEvict(item.Key, item.Value)
				}
			}
		}()
	}
}
