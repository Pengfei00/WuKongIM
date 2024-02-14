package cluster

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/cluster/cluster/clusterconfig/pb"
	"github.com/WuKongIM/WuKongIM/pkg/wkhttp"
	"github.com/WuKongIM/WuKongIM/pkg/wkutil"
	"go.uber.org/zap"
)

func (s *Server) ServerAPI(route *wkhttp.WKHttp, prefix string) {

	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	prefix = strings.TrimSuffix(prefix, "/")

	route.GET(fmt.Sprintf("%s/nodes", prefix), s.clusterInfoGet) // 获取所有节点

	route.GET(fmt.Sprintf("%s/channels/:channel_id/:channel_type/config", prefix), s.channelClusterConfigGet) // 获取频道分布式配置
	route.GET(fmt.Sprintf("%s/slots/:id/config", prefix), s.slotClusterConfigGet)                             // 槽分布式配置

	// route.GET(fmt.Sprintf("%s/channel/clusterinfo", prefix), s.getAllClusterInfo) // 获取所有channel的集群信息
}

func (s *Server) clusterInfoGet(c *wkhttp.Context) {
	cfgServer := s.clusterEventListener.clusterconfigManager.clusterconfigServer
	cfg := cfgServer.ConfigManager().GetConfig()

	leaderId := cfgServer.Leader()

	nodeCfgs := make([]*NodeConfig, 0, len(cfg.Nodes))
	for _, node := range cfg.Nodes {
		nodeCfg := NewNodeConfigFromNode(node)
		nodeCfg.IsLeader = wkutil.BoolToInt(leaderId == node.Id)
		nodeCfg.SlotCount = s.getNodeSlotCount(node.Id, cfg)
		nodeCfgs = append(nodeCfgs, nodeCfg)

	}

	c.JSON(http.StatusOK, NodeConfigTotal{
		Total: len(nodeCfgs),
		Nodes: nodeCfgs,
	})
}

func (s *Server) channelClusterConfigGet(c *wkhttp.Context) {
	channelId := c.Param("channel_id")
	channelTypeStr := c.Param("channel_type")

	channelTypeI64, err := strconv.ParseUint(channelTypeStr, 10, 8)
	if err != nil {
		s.Error("channelTypeStr parse error", zap.Error(err))
		c.ResponseError(err)
		return
	}
	channelType := uint8(channelTypeI64)

	ch, err := s.channelGroupManager.fetchChannel(channelId, channelType)
	if err != nil {
		s.Error("fetchChannel error", zap.Error(err))
		c.ResponseError(err)
		return
	}
	cfg := ch.getClusterConfig()
	slotId := s.getChannelSlotId(channelId)
	slot := s.clusterEventListener.clusterconfigManager.slot(slotId)
	if slot == nil {
		s.Error("slot not found", zap.Uint32("slotId", slotId))
		c.ResponseError(err)
		return

	}
	resp := NewChannelClusterConfigRespFromClusterConfig(slot.Leader, slotId, cfg)
	shardNo := ChannelKey(channelId, channelType)
	lastMsgSeq, lastAppendTime, err := s.opts.MessageLogStorage.LastIndexAndAppendTime(shardNo)
	if err != nil {
		s.Error("LastIndexAndAppendTime error", zap.Error(err))
		c.ResponseError(err)
		return

	}
	resp.MaxMessageSeq = lastMsgSeq
	if lastAppendTime > 0 {
		resp.LastAppendTime = wkutil.ToyyyyMMddHHmm(time.Unix(int64(lastAppendTime/1e9), 0))
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Server) slotClusterConfigGet(c *wkhttp.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		s.Error("id parse error", zap.Error(err))
		c.ResponseError(err)
		return
	}
	slotId := uint32(id)
	shardNo := GetSlotShardNo(slotId)

	slot := s.clusterEventListener.clusterconfigManager.slot(slotId)
	if slot == nil {
		s.Error("slot not found", zap.Uint32("slotId", slotId))
		c.ResponseError(err)
		return
	}

	leaderLogMaxIndex, err := s.getSlotMaxLogIndex(slotId)
	if err != nil {
		s.Error("getSlotMaxLogIndex error", zap.Error(err))
		c.ResponseError(err)
		return
	}
	appliedIdx, err := s.localStorage.getAppliedIndex(shardNo)
	if err != nil {
		s.Error("getAppliedIndex error", zap.Error(err))
		c.ResponseError(err)
		return
	}
	lastIdx, err := s.opts.ShardLogStorage.LastIndex(shardNo)
	if err != nil {
		s.Error("LastIndex error", zap.Error(err))
		c.ResponseError(err)
		return
	}

	cfg := NewSlotClusterConfigRespFromClusterConfig(appliedIdx, lastIdx, leaderLogMaxIndex, slot)
	c.JSON(http.StatusOK, cfg)

}

func (s *Server) getSlotMaxLogIndex(slotId uint32) (uint64, error) {

	slot := s.clusterEventListener.clusterconfigManager.slot(slotId)
	if slot == nil {
		return 0, errors.New("slot not found")
	}

	if slot.Leader == s.opts.NodeID {
		shardNo := GetSlotShardNo(slotId)
		lastIdx, err := s.opts.ShardLogStorage.LastIndex(shardNo)
		if err != nil {
			return 0, err
		}
		return lastIdx, nil
	}

	slotLogResp, err := s.nodeManager.requestSlotLogInfo(slot.Leader, &SlotLogInfoReq{
		SlotIds: []uint32{slotId},
	})
	if err != nil {
		s.Error("requestSlotLogInfo error", zap.Error(err))
		return 0, err
	}
	if len(slotLogResp.Slots) > 0 {
		return slotLogResp.Slots[0].LogIndex, nil
	}
	return 0, nil
}

func (s *Server) getNodeSlotCount(nodeId uint64, cfg *pb.Config) int {
	count := 0
	for _, st := range cfg.Slots {
		if wkutil.ArrayContainsUint64(st.Replicas, nodeId) {
			count++
		}
	}
	return count
}

type NodeConfigTotal struct {
	Total int           `json:"total"` // 总数
	Nodes []*NodeConfig `json:"nodes"`
}

type NodeConfig struct {
	Id            uint64 `json:"id"`                        // 节点ID
	IsLeader      int    `json:"is_leader,omitempty"`       // 是否是leader
	ClusterAddr   string `json:"cluster_addr"`              // 集群地址
	ApiServerAddr string `json:"api_server_addr,omitempty"` // API服务地址
	Online        int    `json:"online,omitempty"`          // 是否在线
	OfflineCount  int    `json:"offline_count,omitempty"`   // 下线次数
	LastOffline   string `json:"last_offline,omitempty"`    // 最后一次下线时间
	AllowVote     int    `json:"allow_vote"`                // 是否允许投票
	SlotCount     int    `json:"slot_count,omitempty"`      // 槽位数量
}

func NewNodeConfigFromNode(n *pb.Node) *NodeConfig {
	// lastOffline format string
	lastOffline := ""
	if n.LastOffline != 0 {
		lastOffline = wkutil.ToyyyyMMddHHmm(time.Unix(n.LastOffline, 0))
	}
	return &NodeConfig{
		Id:            n.Id,
		ClusterAddr:   n.ClusterAddr,
		ApiServerAddr: n.ApiServerAddr,
		Online:        wkutil.BoolToInt(n.Online),
		OfflineCount:  int(n.OfflineCount),
		LastOffline:   lastOffline,
		AllowVote:     wkutil.BoolToInt(n.AllowVote),
	}
}

type ChannelClusterConfigResp struct {
	ChannelID      string   `json:"channel_id"`       // 频道ID
	ChannelType    uint8    `json:"channel_type"`     // 频道类型
	SlotId         uint32   `json:"slot_id"`          // 槽位ID
	SlotLeaderId   uint64   `json:"slot_leader_id"`   // 槽位领导者ID
	ReplicaCount   uint16   `json:"replica_count"`    // 副本数量
	Replicas       []uint64 `json:"replicas"`         // 副本节点ID集合
	LeaderId       uint64   `json:"leader_id"`        // 领导者ID
	Term           uint32   `json:"term"`             // 任期
	MaxMessageSeq  uint64   `json:"max_message_seq"`  // 最大消息序号
	LastAppendTime string   `json:"last_append_time"` // 最后一次追加时间
}

func NewChannelClusterConfigRespFromClusterConfig(slotLeaderId uint64, slotId uint32, cfg *ChannelClusterConfig) *ChannelClusterConfigResp {
	return &ChannelClusterConfigResp{
		ChannelID:    cfg.ChannelID,
		ChannelType:  cfg.ChannelType,
		ReplicaCount: cfg.ReplicaCount,
		Replicas:     cfg.Replicas,
		LeaderId:     cfg.LeaderId,
		Term:         cfg.Term,
		SlotId:       slotId,
		SlotLeaderId: slotLeaderId,
	}
}

type SlotClusterConfigResp struct {
	Id                uint32   `json:"id"`                   // 槽位ID
	LeaderId          uint64   `json:"leader_id"`            // 领导者ID
	Term              uint32   `json:"term"`                 // 任期
	Replicas          []uint64 `json:"replicas"`             // 副本节点ID集合
	ReplicaCount      uint32   `json:"replica_count"`        // 副本数量
	LogMaxIndex       uint64   `json:"log_max_index"`        // 本地日志最大索引
	LeaderLogMaxIndex uint64   `json:"leader_log_max_index"` // 领导者日志最大索引
	AppliedIndex      uint64   `json:"applied_index"`        // 已应用索引
}

func NewSlotClusterConfigRespFromClusterConfig(appliedIdx, logMaxIndex uint64, leaderLogMaxIndex uint64, slot *pb.Slot) *SlotClusterConfigResp {
	return &SlotClusterConfigResp{
		Id:                slot.Id,
		LeaderId:          slot.Leader,
		Term:              slot.Term,
		Replicas:          slot.Replicas,
		ReplicaCount:      slot.ReplicaCount,
		LogMaxIndex:       logMaxIndex,
		LeaderLogMaxIndex: leaderLogMaxIndex,
		AppliedIndex:      appliedIdx,
	}
}
