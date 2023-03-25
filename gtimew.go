package main

import (
	"container/list"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type GTimeWTask struct {
	// delay will be used for calculation round and slot position
	delay time.Duration
	round int
	hash  string
	param interface{}
	do    func(interface{})
}

// GTimeW time wheel
type GTimeW struct {
	// grad is the time wheel graduations
	grad time.Duration
	// ticker is the internal trigger
	ticker *time.Ticker
	// slots store tasks for time wheel
	slots []*list.List

	// pos indicate the time wheel pointer position
	pos int
	// size is the time wheel total slots num
	size int

	// taskPool indicates whether to use a goroutine pool to run tasks
	taskPool *sync.Pool

	taskRecord map[string]int

	stopped int32

	addCh chan *GTimeWTask
	rmCh  chan string
}

// NewGTimeW todo pool
func NewGTimeW(grad time.Duration, size int, wpool *sync.Pool) (*GTimeW, error) {
	if grad <= 0 || size <= 0 {
		return nil, errors.New("graduation and size illegal")
	}
	gtw := &GTimeW{
		grad:  grad,
		slots: make([]*list.List, size),
		pos:   0,
		size:  size,

		taskRecord: make(map[string]int, 8),
		stopped:    0,
		addCh:      make(chan *GTimeWTask),
		rmCh:       make(chan string),
	}
	if wpool != nil {
		gtw.taskPool = wpool
	}
	for i := range gtw.slots {
		gtw.slots[i] = list.New()
	}
	return gtw, nil
}

func (gtw *GTimeW) AddTask(delay time.Duration, taskHash string, do func(interface{}), param interface{}) {
	// delay time must greater than graduation
	if delay.Nanoseconds() < gtw.grad.Nanoseconds() {
		delay = 0
	}
	taskHash = strings.TrimSpace(taskHash)
	if delay < 0 || taskHash == "" || do == nil {
		return
	}
	if atomic.LoadInt32(&gtw.stopped) == 1 {
		return
	}
	// schedule the task right now
	if delay == 0 {
		if gtw.taskPool != nil {
			// todo
			//gtw.taskPool.Submit()
		} else {
			go do(param)
		}
		return
	}
	gtw.addCh <- &GTimeWTask{
		delay: delay,
		hash:  taskHash,
		do:    do,
		param: param,
	}
}

func (gtw *GTimeW) RmTask(taskHash string) {
	taskHash = strings.TrimSpace(taskHash)
	if taskHash == "" {
		return
	}
	if atomic.LoadInt32(&gtw.stopped) == 1 {
		return
	}
	gtw.rmCh <- taskHash
}

func (gtw *GTimeW) Start() {
	gtw.ticker = time.NewTicker(gtw.grad)
	go gtw.startup()
}

func (gtw *GTimeW) Stop() {
	atomic.StoreInt32(&gtw.stopped, 1)
}

func (gtw *GTimeW) startup() {
	for {
		select {
		case <-gtw.ticker.C:
			gtw.forward()
		case task := <-gtw.addCh:
			gtw.add(task)
		case taskHash := <-gtw.rmCh:
			gtw.remove(taskHash)
		}
		// check time wheel whether has been closed
		if atomic.LoadInt32(&gtw.stopped) == 1 {
			// close ticker here to prevent reading in this function
			gtw.ticker.Stop()
			return
		}
	}
}

func (gtw *GTimeW) remove(key string) {
	pos, ok := gtw.taskRecord[key]
	if !ok {
		return
	}
	l := gtw.slots[pos]
	for t := l.Front(); t != nil; {
		task := t.Value.(*GTimeWTask)

		next := t.Next()
		if task.hash == key {
			delete(gtw.taskRecord, key)
			l.Remove(t)
		}
		t = next
	}
}

func (gtw *GTimeW) add(t *GTimeWTask) {
	round, pos := gtw.calculate(t.delay)
	t.round = round

	gtw.slots[pos].PushBack(t)
	gtw.taskRecord[t.hash] = pos
}

// forward one graduation, and scan task list
func (gtw *GTimeW) forward() {
	l := gtw.slots[gtw.pos]
	gtw.schedule(l)
	if gtw.pos == gtw.size-1 {
		gtw.pos = 0
	} else {
		gtw.pos++
	}
}

func (gtw *GTimeW) schedule(l *list.List) {
	for t := l.Front(); t != nil; {
		task := t.Value.(*GTimeWTask)
		// if the task not ready to run
		if task.round > 0 {
			task.round--
			println(task.round)
			t = t.Next()
			continue
		}
		if gtw.taskPool != nil {
			// TODO: use pool to run task
			// taskPool.Submit(task.do, task.param)
		} else {
			go task.do(task.param)
		}
		// store the next task
		next := t.Next()

		// remove the task which has been scheduled
		l.Remove(t)
		// remove task record
		delete(gtw.taskRecord, task.hash)

		// get the next task
		t = next
	}
}

func (gtw *GTimeW) calculate(delay time.Duration) (round, slotPos int) {
	// unify time unit and get slot elapse
	delayNano := delay.Nanoseconds()
	gradNano := gtw.grad.Nanoseconds()

	gradElapse := int(delayNano / gradNano)

	round = gradElapse / gtw.size
	slotPos = (gtw.pos + gradElapse) % gtw.size
	return
}
