package cluster

import (
	"context"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/clusterevent"
	"github.com/WuKongIM/WuKongIM/pkg/clusterevent/pb"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func (s *Server) loopClusterEvent() {
	for {
		select {
		case clusterEvent := <-s.clusterEventManager.Watch():
			s.Debug("收到集群事件", zap.String("clusterEvent", clusterEvent.String()))
			s.handleClusterEvent(clusterEvent)
		case <-s.stopper.ShouldStop():
			return
		}
	}
}

func (s *Server) handleClusterEvent(clusterEvent clusterevent.ClusterEvent) {
	if clusterEvent.SlotEvent != nil {
		s.handleClusterSlotEvent(clusterEvent.SlotEvent)
	}
	if clusterEvent.NodeEvent != nil {
		s.handleClusterNodeEvent(clusterEvent.NodeEvent)
	}

	if clusterEvent.ClusterEventType == pb.ClusterEventType_ClusterEventTypeVersionChange {
		s.handleClusterConfigVersionChange()
	}
}

func (s *Server) handleClusterSlotEvent(slotEvent *pb.SlotEvent) {
	switch slotEvent.EventType {
	case pb.SlotEventType_SlotEventTypeInit: // slot初始化
		s.handleClusterSlotEventInit(slotEvent)
	case pb.SlotEventType_SlotEventTypeElection: // slot选举
		s.handleClusterSlotEventElection(slotEvent)
	}
}

func (s *Server) handleClusterNodeEvent(nodeEvent *pb.NodeEvent) {
	switch nodeEvent.EventType {
	case pb.NodeEventType_NodeEventTypeRequestUpdate: // 请求领导节点更新我的节点信息
		s.handleClusterNodeEventRequestUpdate(nodeEvent)
	case pb.NodeEventType_NodeEventTypeOnlineStatusChange: // 节点在线状态改变
		s.handleClusterNodeEventOnlineChange(nodeEvent)
	}
}

func (s *Server) handleClusterNodeEventRequestUpdate(nodeEvent *pb.NodeEvent) {
	if len(nodeEvent.Node) == 0 {
		return
	}
	node := nodeEvent.Node[0]
	if s.clusterEventManager.IsNodeLeader() { // 当前节点是领导节点，那么直接修改节点信息
		s.clusterEventManager.UpdateNode(node)
		return
	}
	// 当前节点不是领导节点，那么需要向领导节点请求更新
	if s.leaderID.Load() != 0 {
		err := s.nodeManager.requestNodeUpdate(s.cancelCtx, s.leaderID.Load(), node)
		if err != nil {
			s.Error("requestNodeUpdate is failed", zap.Error(err))
		}
	}
}

func (s *Server) handleClusterNodeEventOnlineChange(nodeEvent *pb.NodeEvent) {
	if len(nodeEvent.Node) == 0 {
		return
	}
	if !s.clusterEventManager.IsNodeLeader() {
		return
	}
	for _, node := range nodeEvent.Node {
		s.clusterEventManager.SetNodeOnlineNoSave(node.Id, node.Online)
	}
	s.clusterEventManager.SaveAndVersionInc()
}

func (s *Server) handleClusterConfigVersionChange() {
	slots := s.clusterEventManager.GetSlots()
	if len(slots) == 0 {
		return
	}
	err := s.addOrUpdateSlots(slots)
	if err != nil {
		s.Error("addOrUpdateSlots is failed", zap.Error(err))
	}
}

func (s *Server) addOrUpdateSlots(slots []*pb.Slot) error {
	var err error
	for _, st := range slots {
		slot := s.slotManager.GetSlot(st.Id)
		if slot == nil {
			slot, err = s.newSlot(st)
			if err != nil {
				s.Error("slot init failed", zap.Error(err))
				return err
			}
			s.slotManager.AddSlot(slot)
		} else {
			if slot.LeaderID() != st.Leader {
				slot.SetLeaderID(st.Leader)
			}
		}
	}
	return nil
}

// 处理slot初始化
func (s *Server) handleClusterSlotEventInit(slotEvent *pb.SlotEvent) {
	if len(slotEvent.Slots) == 0 {
		return
	}
	if !s.clusterEventManager.IsNodeLeader() {
		s.clusterEventManager.SetSlotIsInit(true)
		return
	}
	for _, st := range slotEvent.Slots {
		s.clusterEventManager.AddOrUpdateSlotNoSave(st)

	}
	err := s.addOrUpdateSlots(slotEvent.Slots)
	if err != nil {
		s.Error("addOrUpdateSlots is failed", zap.Error(err))
	}
	s.clusterEventManager.SaveAndVersionInc()
	s.clusterEventManager.SetSlotIsInit(true)

}

// 处理slot选举
func (s *Server) handleClusterSlotEventElection(slotEvent *pb.SlotEvent) {
	if len(slotEvent.SlotIDs) == 0 {
		return
	}

	// 计算slot分布的节点
	slotNodeMap := map[uint64][]uint32{}
	slots := s.clusterEventManager.GetSlots()
	for _, slot := range slots {
		for _, replicaNodeID := range slot.Replicas {
			if replicaNodeID == s.opts.NodeID {
				slotNodeMap[replicaNodeID] = append(slotNodeMap[replicaNodeID], slot.Id)
			} else {
				node := s.nodeManager.getNode(replicaNodeID)
				if node != nil && s.clusterEventManager.NodeIsOnline(replicaNodeID) {
					slotNodeMap[replicaNodeID] = append(slotNodeMap[replicaNodeID], slot.Id)
				}
			}
		}
	}
	if len(slotNodeMap) == 0 {
		s.Debug("没有可用的节点")
		return
	}

	// 发送上报slot信息的请求
	slotInfoReportResps := make([]*SlotLogInfoReportResponse, 0)
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	requestGroup, ctx := errgroup.WithContext(timeoutCtx)

	for nodeID, slotIDs := range slotNodeMap {
		if nodeID == s.opts.NodeID { // 本节点
			slotInfos, err := s.getSlotInfosFromLocalNode(slotIDs)
			if err != nil {
				s.Error("getSlotInfosFromLocalNode is failed", zap.Error(err))
				cancel()
				return
			}
			slotInfoReportResps = append(slotInfoReportResps, &SlotLogInfoReportResponse{
				NodeID:    nodeID,
				SlotInfos: slotInfos,
			})
			continue
		}
		if !s.clusterEventManager.NodeIsOnline(nodeID) { // 必需在线
			continue
		}
		requestGroup.Go(func(nID uint64, slotIDs []uint32) func() error {
			return func() error {
				select {
				case <-ctx.Done():
				default:
					req := &SlotLogInfoReportRequest{
						SlotIDs: slotIDs,
					}
					resp, err := s.nodeManager.requestSlotLogInfo(s.cancelCtx, nID, req)
					if err != nil {
						return err
					}
					slotInfoReportResps = append(slotInfoReportResps, resp)
				}
				return nil
			}
		}(nodeID, slotIDs))
	}

	if err := requestGroup.Wait(); err != nil {
		s.Error("requestSlotLogInfo is failed", zap.Error(err))
		cancel()
		return
	}
	cancel()

	if len(slotInfoReportResps) == 0 {
		return
	}

	// 收集slot信息
	nodeSlotLogIndexMap := s.convertSlotInfoReportRespsToNodeSlotMap(slotInfoReportResps)

	// 获取slot的领导者
	slotLeaderMap := s.getSlotLeaderByNodeSlotMap(nodeSlotLogIndexMap, slotEvent.SlotIDs)

	if len(slotLeaderMap) == 0 {
		s.Warn("没有选举出任何槽领导者！！！", zap.Uint32s("slotIDs", slotEvent.SlotIDs))
		return
	}

	for slotID, leaderNodeID := range slotLeaderMap {
		s.Debug("slot选举", zap.Uint32("slotID", slotID), zap.Uint64("leaderNodeID", leaderNodeID))
		s.clusterEventManager.UpdateSlotLeaderNoSave(slotID, leaderNodeID)
	}
	s.clusterEventManager.SaveAndVersionInc()

}

// 将slotInfoReportResps转换成 各个节点的slot对应的logIndex
func (s *Server) convertSlotInfoReportRespsToNodeSlotMap(slotInfoReportResps []*SlotLogInfoReportResponse) map[uint64]map[uint32]uint64 {
	nodeSlotMap := map[uint64]map[uint32]uint64{} // nodeID -> slotID -> logIndex
	for _, slotInfoReportResp := range slotInfoReportResps {
		slotLogMap := nodeSlotMap[slotInfoReportResp.NodeID]
		if slotLogMap != nil {
			for _, slotInfo := range slotInfoReportResp.SlotInfos {
				slotLogMap[slotInfo.SlotID] = slotInfo.LogIndex
			}
		} else {
			slotLogMap = map[uint32]uint64{}
			for _, slotInfo := range slotInfoReportResp.SlotInfos {
				slotLogMap[slotInfo.SlotID] = slotInfo.LogIndex
			}
			nodeSlotMap[slotInfoReportResp.NodeID] = slotLogMap
		}
	}
	return nodeSlotMap
}

// 根据nodeSlotLogIndexMap信息分析出，指定的sslotIDs的领导者
func (s *Server) getSlotLeaderByNodeSlotMap(nodeSlotLogIndexMap map[uint64]map[uint32]uint64, slotIDs []uint32) map[uint32]uint64 {
	slotLeaderMap := map[uint32]uint64{}
	for _, slotID := range slotIDs {
		leaderNodeID := uint64(0)
		var leaderLogIndex uint64
		for nodeID, slotLogMap := range nodeSlotLogIndexMap {
			logIndex, ok := slotLogMap[slotID]
			if ok {
				if leaderNodeID == 0 {
					leaderNodeID = nodeID
					leaderLogIndex = logIndex
				} else {
					if leaderLogIndex < logIndex {
						leaderNodeID = nodeID
						leaderLogIndex = logIndex
					}
				}
			}
		}
		if leaderNodeID != 0 {
			slotLeaderMap[slotID] = leaderNodeID
		}
	}
	return slotLeaderMap
}

func (s *Server) getSlotInfosFromLocalNode(slotIDs []uint32) ([]*SlotInfo, error) {
	slotInfos := make([]*SlotInfo, 0)
	for _, slotID := range slotIDs {
		slot := s.slotManager.GetSlot(uint32(slotID))
		if slot != nil {
			lastLogIndex, err := slot.LastLogIndex()
			if err != nil {
				return nil, err
			}
			slotInfos = append(slotInfos, &SlotInfo{
				SlotID:   uint32(slotID),
				LogIndex: lastLogIndex,
			})
		}
	}
	return slotInfos, nil
}