/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package comm

import (
	"sync"
	"sync/atomic"

	"github.com/hyperledger/fabric/gossip/common"
)

// ChannelDeMultiplexer is a struct that can receive channel registrations (AddChannel)
// and publications (DeMultiplex) and it broadcasts the publications to registrations
// according to their predicate
type ChannelDeMultiplexer struct {
	channels []*channel
	lock     *sync.RWMutex
	closed   int32
}

// NewChannelDemultiplexer creates a new ChannelDeMultiplexer
func NewChannelDemultiplexer() *ChannelDeMultiplexer {
	return &ChannelDeMultiplexer{
		channels: make([]*channel, 0),
		lock:     &sync.RWMutex{},
		closed:   int32(0),
	}
}

type channel struct {
	pred common.MessageAcceptor
	ch   chan interface{}
}

func (m *ChannelDeMultiplexer) isClosed() bool {
	return atomic.LoadInt32(&m.closed) == int32(1)
}

// Close closes this channel, which makes all channels registered before
// to close as well.
func (m *ChannelDeMultiplexer) Close() {
	defer func() {
		// recover closing an already closed channel
		recover()
	}()
	atomic.StoreInt32(&m.closed, int32(1))
	m.lock.Lock()
	defer m.lock.Unlock()
	for _, ch := range m.channels {
		close(ch.ch)
	}
}

// AddChannel registers a channel with a certain predicate
func (m *ChannelDeMultiplexer) AddChannel(predicate common.MessageAcceptor) chan interface{} {
	m.lock.Lock()
	defer m.lock.Unlock()
	ch := &channel{ch: make(chan interface{}, 10), pred: predicate}
	m.channels = append(m.channels, ch)
	return ch.ch
}

// DeMultiplex broadcasts the message to all channels that were returned
// by AddChannel calls and that hold the respected predicates.
func (m *ChannelDeMultiplexer) DeMultiplex(msg interface{}) {
	defer func() {
		recover()
	}() // recover from sending on a closed channel

	if m.isClosed() {
		return
	}

	m.lock.RLock()
	channels := m.channels
	m.lock.RUnlock()

	for _, ch := range channels {
		if ch.pred(msg) {
			ch.ch <- msg
		}
	}
}
