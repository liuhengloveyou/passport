package cache

import (
	"container/list"
	"sync"
	"time"
)

type data struct {
	key       interface{}
	val       interface{}
	expiredAt int64
}

type ExpiredMap struct {
	m        map[interface{}]*data
	timeList *list.List
	lck      *sync.Mutex
}

func NewExpiredMap() *ExpiredMap {
	e := ExpiredMap{
		m:        make(map[interface{}]*data),
		timeList: list.New(),
		lck:      new(sync.Mutex),
	}

	go e.run()

	return &e
}

func (e *ExpiredMap) run() {
	for {
		time.Sleep(time.Second)

		now := time.Now().Unix()
		for {
			ele := e.timeList.Front()
			if ele == nil || ele.Value == nil {
				break
			}

			if ele.Value.(data).expiredAt >= now {
				break
			}

			e.lck.Lock()
			e.timeList.Remove(ele)
			delete(e.m, ele.Value.(data).key)
			e.lck.Unlock()
		}
	}
}

func (e *ExpiredMap) Set(key, value interface{}, aliveSecond int64) {
	if aliveSecond <= 0 {
		return
	}

	e.lck.Lock()
	defer e.lck.Unlock()

	expiredAt := time.Now().Unix() + aliveSecond
	tmpData := &data{
		key:       key,
		val:       value,
		expiredAt: expiredAt,
	}
	e.timeList.PushBack(tmpData)
	e.m[key] = tmpData
}

func (e *ExpiredMap) Get(key interface{}) (found bool, value interface{}) {
	e.lck.Lock()
	defer e.lck.Unlock()

	if found = e.checkDeleteKey(key); !found {
		return
	}

	value = e.m[key].val

	return
}

func (e *ExpiredMap) Delete(key interface{}) {
	e.lck.Lock()
	delete(e.m, key)
	e.lck.Unlock()
}

func (e *ExpiredMap) Length() int {
	e.lck.Lock()
	defer e.lck.Unlock()

	return len(e.m)
}

func (e *ExpiredMap) TTL(key interface{}) int64 {
	e.lck.Lock()
	defer e.lck.Unlock()

	if !e.checkDeleteKey(key) {
		return -1
	}

	return e.m[key].expiredAt - time.Now().Unix()
}

func (e *ExpiredMap) Clear() {
	e.lck.Lock()
	defer e.lck.Unlock()

	e.m = make(map[interface{}]*data)
	e.timeList = list.New()
}

func (e *ExpiredMap) DoForEach(handler func(interface{}, interface{})) {
	e.lck.Lock()
	defer e.lck.Unlock()
	for k, v := range e.m {
		if !e.checkDeleteKey(k) {
			continue
		}
		handler(k, v)
	}
}

func (e *ExpiredMap) DoForEachWithBreak(handler func(interface{}, interface{}) bool) {
	e.lck.Lock()
	defer e.lck.Unlock()

	for k, v := range e.m {
		if !e.checkDeleteKey(k) {
			continue
		}
		if handler(k, v) {
			break
		}
	}
}

func (e *ExpiredMap) checkDeleteKey(key interface{}) bool {
	if val, found := e.m[key]; found {
		if val.expiredAt <= time.Now().Unix() {
			delete(e.m, key)
			return false
		}
		return true
	}

	return false
}
