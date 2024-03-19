// Copyright 2017-2019 Lei Ni (nilei81@gmail.com) and other contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"sync"

	"github.com/WuKongIM/WuKongIM/pkg/cluster/replica"
	"github.com/WuKongIM/WuKongIM/pkg/wklog"
	"go.uber.org/zap"
)

type MessageQueue struct {
	left          []Message
	right         []Message
	nodrop        []Message
	rl            *RateLimiter // 速率限制
	size          uint64
	lazyFreeCycle uint64
	mu            sync.Mutex

	cycle uint64

	idx    uint64
	oldIdx uint64

	leftInWrite bool
	stopped     bool

	wklog.Log
}

func NewMessageQueue(size uint64,
	ch bool, lazyFreeCycle uint64, maxMemorySize uint64) *MessageQueue {
	q := &MessageQueue{
		rl:            NewRateLimiter(maxMemorySize),
		size:          size,
		lazyFreeCycle: lazyFreeCycle,
		left:          make([]Message, size),
		right:         make([]Message, size),
		nodrop:        make([]Message, 0),
	}
	return q
}

func (q *MessageQueue) targetQueue() []Message {
	var t []Message
	if q.leftInWrite {
		t = q.left
	} else {
		t = q.right
	}
	return t
}

// Add adds the specified message to the queue.
func (q *MessageQueue) Add(msg Message) (bool, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.idx >= q.size {
		return false, q.stopped
	}
	if q.stopped {
		return false, true
	}
	if !q.tryAdd(msg) {
		return false, false
	}
	w := q.targetQueue()
	w[q.idx] = msg
	q.idx++
	return true, false
}

// MustAdd adds the specified message to the queue.
func (q *MessageQueue) MustAdd(msg Message) bool {

	q.mu.Lock()
	defer q.mu.Unlock()
	if q.stopped {
		return false
	}
	q.nodrop = append(q.nodrop, msg)
	return true
}

func (q *MessageQueue) tryAdd(msg Message) bool {
	if !q.rl.Enabled() {
		return true
	}
	if q.rl.RateLimited() {
		q.Warn("rate limited dropped a Replicate msg", zap.String("shardNo", msg.ShardNo))
		return false
	}
	q.rl.Increase(uint64(replica.LogsSize(msg.Logs)))
	return true
}

func (q *MessageQueue) gc() {
	if q.lazyFreeCycle > 0 {
		oldq := q.targetQueue()
		if q.lazyFreeCycle == 1 {
			for i := uint64(0); i < q.oldIdx; i++ {
				oldq[i].Logs = nil
			}
		} else if q.cycle%q.lazyFreeCycle == 0 {
			for i := uint64(0); i < q.size; i++ {
				oldq[i].Logs = nil
			}
		}
	}
}

// Get returns everything current in the queue.
func (q *MessageQueue) Get() []Message {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.cycle++
	sz := q.idx
	q.idx = 0
	t := q.targetQueue()
	q.leftInWrite = !q.leftInWrite
	q.gc()
	q.oldIdx = sz
	if q.rl.Enabled() {
		q.rl.Set(0)
	}
	if len(q.nodrop) == 0 {
		return t[:sz]
	}

	var result []Message
	if len(q.nodrop) > 0 {
		ssm := q.nodrop
		q.nodrop = make([]Message, 0)
		result = append(result, ssm...)
	}
	return append(result, t[:sz]...)
}