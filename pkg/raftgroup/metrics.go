package raftgroup

import (
	"fmt"

	"go.uber.org/atomic"
)

type metrics struct {
	recvMsgCount         atomic.Uint64 // 接受客户端发送的消息数量
	recvMsgBytes         atomic.Uint64 // 接受客户端发送的消息字节数
	sendMsgCount         atomic.Uint64 // 投递给客户端的消息数量
	sendMsgBytes         atomic.Uint64 // 投递给客户端的消息字节数
	campaignTimeoutCount atomic.Uint64 // 竞选超时次数
}

func newMetrics() *metrics {
	return &metrics{}
}

func (m *metrics) recvMsgCountAdd(v uint64) uint64 {
	return m.recvMsgCount.Add(v)
}

func (m *metrics) recvMsgCountSub(v uint64) uint64 {
	return m.recvMsgCount.Add(-v)
}

func (m *metrics) recvMsgBytesAdd(v uint64) {
	m.recvMsgBytes.Add(v)
}

func (m *metrics) recvMsgBytesSub(v uint64) {
	m.recvMsgBytes.Add(-v)
}

func (m *metrics) sendMsgCountAdd(v uint64) uint64 {
	return m.sendMsgCount.Add(v)
}

func (m *metrics) sendMsgCountSub(v uint64) uint64 {
	return m.sendMsgCount.Add(-v)
}

func (m *metrics) sendMsgBytesAdd(v uint64) {
	m.sendMsgBytes.Add(v)
}

func (m *metrics) sendMsgBytesSub(v uint64) {
	m.sendMsgBytes.Add(-v)
}

func (m *metrics) campaignTimeoutCountAdd(v uint64) {
	m.campaignTimeoutCount.Add(v)
}

func (m *metrics) printMetrics(prefix string) {
	recvMsgCount := m.recvMsgCount.Load()
	recvMsgBytes := m.recvMsgBytes.Load()
	sendMsgCount := m.sendMsgCount.Load()
	sendMsgBytes := m.sendMsgBytes.Load()
	campaignTimeoutCount := m.campaignTimeoutCount.Load()

	fmt.Printf("[%s] campaignTimeoutCount: %d  recvMsgCount: %d, recvMsgBytes: %d, sendMsgCount: %d, sendMsgBytes: %d \n", prefix, campaignTimeoutCount, recvMsgCount, recvMsgBytes, sendMsgCount, sendMsgBytes)
}
