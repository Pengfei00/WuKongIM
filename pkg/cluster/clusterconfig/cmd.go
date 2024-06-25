package clusterconfig

import (
	"github.com/WuKongIM/WuKongIM/pkg/cluster/clusterconfig/pb"
	"github.com/WuKongIM/WuKongIM/pkg/wkutil"
	wkproto "github.com/WuKongIM/WuKongIMGoProto"
)

type CMDType uint16

const (
	CMDTypeUnknown                   CMDType = iota
	CMDTypeConfigInit                        // 初始化配置
	CMDTypeConfigApiServerAddrChange         // api服务地址变更
	CMDTypeNodeJoin                          // 节点加入
	CMDTypeNodeJoining                       // 节点加入中
	CMDTypeNodeJoined                        // 节点已加入
	CMDTypeNodeOnlineStatusChange            // 节点在线状态改变
	CMDTypeSlotMigrate                       // 槽迁移
	CMDTypeSlotUpdate                        // 槽更新
	CMDTypeNodeStatusChange                  // 节点状态改变

)

func (c CMDType) Uint16() uint16 {
	return uint16(c)
}

func (c CMDType) String() string {
	switch c {
	case CMDTypeConfigInit:
		return "CMDTypeConfigInit"
	case CMDTypeConfigApiServerAddrChange:
		return "CMDTypeConfigApiServerAddrChange"
	case CMDTypeNodeJoin:
		return "CMDTypeNodeJoin"
	case CMDTypeNodeJoining:
		return "CMDTypeNodeJoining"
	case CMDTypeNodeJoined:
		return "CMDTypeNodeJoined"
	case CMDTypeNodeOnlineStatusChange:
		return "CMDTypeNodeOnlineStatusChange"
	case CMDTypeSlotMigrate:
		return "CMDTypeSlotMigrate"
	case CMDTypeSlotUpdate:
		return "CMDTypeSlotUpdate"
	case CMDTypeNodeStatusChange:
		return "CMDTypeNodeStatusChange"
	}
	return "CMDTypeUnknown"
}

type CMD struct {
	CmdType CMDType
	Data    []byte
	version uint16 // 数据协议版本

}

func NewCMD(cmdType CMDType, data []byte) *CMD {
	return &CMD{
		CmdType: cmdType,
		Data:    data,
	}
}

func (c *CMD) Marshal() ([]byte, error) {
	c.version = 1
	enc := wkproto.NewEncoder()
	defer enc.End()
	enc.WriteUint16(c.version)
	enc.WriteUint16(c.CmdType.Uint16())
	enc.WriteBytes(c.Data)
	return enc.Bytes(), nil

}

func (c *CMD) Unmarshal(data []byte) error {
	dec := wkproto.NewDecoder(data)
	var err error
	if c.version, err = dec.Uint16(); err != nil {
		return err
	}
	var cmdType uint16
	if cmdType, err = dec.Uint16(); err != nil {
		return err
	}
	c.CmdType = CMDType(cmdType)
	if c.Data, err = dec.BinaryAll(); err != nil {
		return err
	}
	return nil
}

func EncodeApiServerAddrChange(nodeId uint64, apiServerAddr string) ([]byte, error) {
	enc := wkproto.NewEncoder()
	defer enc.End()
	enc.WriteUint64(nodeId)
	enc.WriteString(apiServerAddr)
	return enc.Bytes(), nil
}

func DecodeApiServerAddrChange(data []byte) (uint64, string, error) {
	dec := wkproto.NewDecoder(data)
	var err error
	var nodeId uint64
	if nodeId, err = dec.Uint64(); err != nil {
		return 0, "", err
	}
	apiServerAddr, err := dec.String()
	return nodeId, apiServerAddr, err
}

func EncodeNodeOnlineStatusChange(nodeId uint64, online bool) ([]byte, error) {
	enc := wkproto.NewEncoder()
	defer enc.End()
	enc.WriteUint64(nodeId)
	enc.WriteUint8(wkutil.BoolToUint8(online))
	return enc.Bytes(), nil
}

func DecodeNodeOnlineStatusChange(data []byte) (uint64, bool, error) {
	dec := wkproto.NewDecoder(data)
	var err error
	var nodeId uint64
	if nodeId, err = dec.Uint64(); err != nil {
		return 0, false, err
	}
	online, err := dec.Uint8()
	return nodeId, wkutil.Uint8ToBool(online), err
}

func EncodeMigrateSlot(slotId uint32, fromNodeId, toNodeId uint64) ([]byte, error) {
	enc := wkproto.NewEncoder()
	defer enc.End()

	enc.WriteUint32(slotId)
	enc.WriteUint64(fromNodeId)
	enc.WriteUint64(toNodeId)

	return enc.Bytes(), nil
}

func DecodeMigrateSlot(data []byte) (slotId uint32, fromNodeId, toNodeId uint64, err error) {
	dec := wkproto.NewDecoder(data)
	if slotId, err = dec.Uint32(); err != nil {
		return
	}

	if fromNodeId, err = dec.Uint64(); err != nil {
		return
	}

	if toNodeId, err = dec.Uint64(); err != nil {
		return
	}

	return
}

func EncodeNodeStatusChange(nodeId uint64, status pb.NodeStatus) ([]byte, error) {
	enc := wkproto.NewEncoder()
	defer enc.End()
	enc.WriteUint64(nodeId)
	enc.WriteUint32(uint32(status))
	return enc.Bytes(), nil
}

func DecodeNodeStatusChange(data []byte) (uint64, pb.NodeStatus, error) {
	dec := wkproto.NewDecoder(data)
	var err error
	var nodeId uint64
	if nodeId, err = dec.Uint64(); err != nil {
		return 0, pb.NodeStatus_NodeStatusUnkown, err
	}
	status, err := dec.Uint32()
	return nodeId, pb.NodeStatus(status), err
}

func EncodeNodeJoined(nodeId uint64, slots []*pb.Slot) ([]byte, error) {
	enc := wkproto.NewEncoder()
	defer enc.End()
	enc.WriteUint64(nodeId)
	for _, slot := range slots {
		data, err := slot.Marshal()
		if err != nil {
			return nil, err
		}
		enc.WriteBinary(data)
	}
	return enc.Bytes(), nil
}

func DecodeNodeJoined(data []byte) (uint64, []*pb.Slot, error) {
	dec := wkproto.NewDecoder(data)
	var err error
	var nodeId uint64
	if nodeId, err = dec.Uint64(); err != nil {
		return 0, nil, err
	}
	var slots []*pb.Slot
	for dec.Len() > 0 {
		data, err := dec.Binary()
		if err != nil {
			return 0, nil, err
		}
		slot := &pb.Slot{}
		if err := slot.Unmarshal(data); err != nil {
			return 0, nil, err
		}
		slots = append(slots, slot)
	}
	return nodeId, slots, nil
}
