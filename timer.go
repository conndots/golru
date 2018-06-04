package golru

import "time"

type Timer interface {
	NowNano() int64
}

type GoTimer struct {}

func (t *GoTimer) NowNano() int64 {
	return time.Now().UnixNano()
}
