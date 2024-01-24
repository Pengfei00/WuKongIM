package cluster

import (
	"sync"
	"testing"
	"time"

	replica "github.com/WuKongIM/WuKongIM/pkg/cluster/replica2"
	"github.com/stretchr/testify/assert"
)

func TestChannelReady(t *testing.T) {
	channelID := "test"
	channelType := uint8(2)
	opts := NewOptions()
	opts.NodeID = 1
	opts.ShardLogStorage = NewMemoryShardLogStorage()
	ch := newChannel(&ChannelClusterConfig{
		ChannelID:   channelID,
		ChannelType: channelType,
		Replicas:    []uint64{1, 2, 3},
	}, 0, opts)

	err := ch.appointLeader(2, 1) // 任命为领导

	assert.NoError(t, err)

	has := ch.hasReady()
	assert.True(t, has)
	rd := ch.ready()
	msgs := rd.Messages
	has, _ = getMessageByType(replica.MsgAppointLeaderResp, msgs)
	assert.True(t, has)

}

func TestChannelLeaderPropose(t *testing.T) {
	channelID := "test"
	channelType := uint8(2)
	opts := NewOptions()
	opts.NodeID = 1
	opts.ShardLogStorage = NewMemoryShardLogStorage()
	ch := newChannel(&ChannelClusterConfig{
		ChannelID:   channelID,
		ChannelType: channelType,
		Replicas:    []uint64{1, 2, 3},
	}, 0, opts)

	err := ch.appointLeader(2, 1) // 任命为领导

	assert.NoError(t, err)

	err = ch.propose([]byte("hello"))
	assert.NoError(t, err)

	has := ch.hasReady()
	assert.True(t, has)

	rd := ch.ready()
	msgs := rd.Messages
	has, _ = getMessageByType(replica.MsgNotifySync, msgs)
	assert.True(t, has)

}

func TestChannelLeaderCommitLog(t *testing.T) {
	channelID := "test"
	channelType := uint8(2)
	opts := NewOptions()
	opts.NodeID = 1
	opts.ShardLogStorage = NewMemoryShardLogStorage()
	ch := newChannel(&ChannelClusterConfig{
		ChannelID:   channelID,
		ChannelType: channelType,
		Replicas:    []uint64{1, 2, 3},
	}, 0, opts)

	err := ch.appointLeader(2, 1) // 任命为领导
	assert.NoError(t, err)

	err = ch.propose([]byte("hello"))
	assert.NoError(t, err)

	err = ch.propose([]byte("hello2"))
	assert.NoError(t, err)

	err = ch.stepLock(replica.Message{
		MsgType: replica.MsgSync,
		From:    2,
		To:      1,
		Term:    1,
		Index:   2,
	})
	assert.NoError(t, err)

}

func TestChannelFollowSync(t *testing.T) {
	channelID := "test"
	channelType := uint8(2)
	opts := NewOptions()
	opts.NodeID = 1
	opts.ShardLogStorage = NewMemoryShardLogStorage()
	ch := newChannel(&ChannelClusterConfig{
		ChannelID:   channelID,
		ChannelType: channelType,
		Replicas:    []uint64{1, 2, 3},
	}, 0, opts)

	err := ch.stepLock(replica.Message{
		MsgType: replica.MsgNotifySync,
		From:    2,
		To:      1,
		Term:    1,
	})
	assert.NoError(t, err)

	has := ch.hasReady()
	assert.True(t, has)

	rd := ch.ready()
	has, _ = getMessageByType(replica.MsgSync, rd.Messages)
	assert.True(t, has)

}

func TestChannelFollowSyncResp(t *testing.T) {
	channelID := "test"
	channelType := uint8(2)
	opts := NewOptions()
	opts.NodeID = 1
	opts.ShardLogStorage = NewMemoryShardLogStorage()
	ch := newChannel(&ChannelClusterConfig{
		ChannelID:   channelID,
		ChannelType: channelType,
		Replicas:    []uint64{1, 2, 3},
	}, 0, opts)

	err := ch.stepLock(replica.Message{
		MsgType: replica.MsgSyncResp,
		From:    2,
		To:      1,
		Term:    1,
		Logs:    []replica.Log{{Index: 1, Term: 1, Data: []byte("hello")}},
	})
	assert.NoError(t, err)

	rd := ch.ready()

	msgs := rd.Messages
	assert.Equal(t, 0, len(msgs))

}

func TestChannelProposeAndWaitCommit(t *testing.T) {
	channelID := "test"
	channelType := uint8(2)
	opts := NewOptions()
	opts.NodeID = 1
	opts.ShardLogStorage = NewMemoryShardLogStorage()
	ch := newChannel(&ChannelClusterConfig{
		ChannelID:   channelID,
		ChannelType: channelType,
		Replicas:    []uint64{1, 2, 3},
	}, 0, opts)

	err := ch.appointLeader(10001, 1) // 任命为领导
	assert.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		_, err = ch.proposeAndWaitCommit([]byte("hello"), time.Second*5)
		assert.NoError(t, err)

		wg.Done()
	}()

	time.Sleep(time.Millisecond * 10) // 等待propose

	// ---------- commit ----------
	// replica 2 sync
	err = ch.stepLock(replica.Message{
		MsgType: replica.MsgSync,
		From:    2,
		Term:    1,
		Index:   2,
		To:      1,
	})
	assert.NoError(t, err)

	rd := ch.ready()

	has, applyMsg := getMessageByType(replica.MsgApplyLogsReq, rd.Messages)
	assert.True(t, has)

	ch.handleLocalMsg(applyMsg)

	wg.Wait()

}

func getMessageByType(ty replica.MsgType, messages []replica.Message) (bool, replica.Message) {
	for _, msg := range messages {
		if msg.MsgType == ty {
			return true, msg
		}
	}
	return false, replica.Message{}
}