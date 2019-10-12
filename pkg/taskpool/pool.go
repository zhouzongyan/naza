// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/naza
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package taskpool

import (
	"container/list"
	"sync"
	"sync/atomic"
)

type pool struct {
	idleWorkerNum int32
	busyWorkerNum int32

	m              sync.Mutex
	idleWorkerList *list.List
}

func (p *pool) Go(task Task) {
	var w *Worker
	p.m.Lock()
	e := p.idleWorkerList.Front()
	if e != nil {
		w = e.Value.(*Worker)
		p.idleWorkerList.Remove(e)
		atomic.AddInt32(&p.idleWorkerNum, -1)
		atomic.AddInt32(&p.busyWorkerNum, 1)
	}
	p.m.Unlock()
	if w == nil {
		w = NewWorker(p)
		w.Start()
		atomic.AddInt32(&p.busyWorkerNum, 1)
	}
	w.Go(task)
}

func (p *pool) KillIdleWorkers() {
	p.m.Lock()
	n := p.idleWorkerList.Len()
	for e := p.idleWorkerList.Front(); e != nil; e = e.Next() {
		w := e.Value.(*Worker)
		w.Stop()
		p.idleWorkerList.Remove(e)
	}
	atomic.AddInt32(&p.idleWorkerNum, int32(-n))
	p.m.Unlock()
}

func (p *pool) Status() (idleWorkerNum int, busyWorkerNum int) {
	idleWorkerNum = int(atomic.LoadInt32(&p.idleWorkerNum))
	busyWorkerNum = int(atomic.LoadInt32(&p.busyWorkerNum))
	return
}

func (p *pool) markIdle(w *Worker) {
	p.m.Lock()
	atomic.AddInt32(&p.idleWorkerNum, 1)
	atomic.AddInt32(&p.busyWorkerNum, -1)
	p.idleWorkerList.PushBack(w)
	p.m.Unlock()
}
