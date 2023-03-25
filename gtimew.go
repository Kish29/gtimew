/*
   Copyright [yyyy] [name of copyright owner]

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"container/list"
	"strings"
	"sync/atomic"
	"time"
)

const twSize = 60

type GTimeWTask struct {
	// delaySec will be used for calculation round and slot position
	delaySec int
	round    int
	hash     string
	param    interface{}
	do       func(interface{})
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
	// TODO: worker pool
	//taskPool *sync.Pool

	taskRecord map[string]int

	stopped int32

	addCh chan *GTimeWTask
	rmCh  chan string
}

func NewGTimeW() *GTimeW {
	gtw := &GTimeW{
		grad:  time.Second,
		slots: make([]*list.List, twSize),
		pos:   0,
		size:  twSize,

		taskRecord: make(map[string]int, 8),
		stopped:    0,
		addCh:      make(chan *GTimeWTask),
		rmCh:       make(chan string),
	}
	for i := range gtw.slots {
		gtw.slots[i] = list.New()
	}
	return gtw
}

func (gtw *GTimeW) AddTask(delaySec int, taskHash string, do func(interface{}), param interface{}) {
	taskHash = strings.TrimSpace(taskHash)
	if delaySec < 0 || taskHash == "" || do == nil {
		return
	}
	if atomic.LoadInt32(&gtw.stopped) == 1 {
		return
	}
	// schedule the task right now
	if delaySec == 0 {
		//if gtw.taskPool != nil {
		// todo
		//gtw.taskPool.Submit()
		//} else {
		go do(param)
		//}
		return
	}
	gtw.addCh <- &GTimeWTask{
		delaySec: delaySec,
		hash:     taskHash,
		do:       do,
		param:    param,
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
	round, pos := gtw.calculate(t.delaySec)
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
			t = t.Next()
			continue
		}
		//if gtw.taskPool != nil {
		// TODO: use pool to run task
		// taskPool.Submit(task.do, task.param)
		//} else {
		go task.do(task.param)
		//}
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

func (gtw *GTimeW) calculate(delaySec int) (round, slotPos int) {
	round = delaySec / gtw.size
	slotPos = (gtw.pos + delaySec) % gtw.size
	if slotPos > 0 {
		slotPos--
	}
	return
}
