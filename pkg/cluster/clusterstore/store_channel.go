package clusterstore

import (
	"github.com/WuKongIM/WuKongIM/pkg/wkdb"
)

// AddSubscribers 添加订阅者
func (s *Store) AddSubscribers(channelId string, channelType uint8, subscribers []string) error {
	data := EncodeSubscribers(channelId, channelType, subscribers)
	cmd := NewCMD(CMDAddSubscribers, data)
	cmdData, err := cmd.Marshal()
	if err != nil {
		return err
	}
	slotId := s.opts.GetSlotId(channelId)
	_, err = s.opts.Cluster.ProposeDataToSlot(s.ctx, slotId, cmdData)
	return err
}

func (s *Store) ExistSubscriber(channelId string, channelType uint8, uid string) (bool, error) {
	return s.wdb.ExistSubscriber(channelId, channelType, uid)
}

// RemoveSubscribers 移除订阅者
func (s *Store) RemoveSubscribers(channelId string, channelType uint8, subscribers []string) error {
	data := EncodeSubscribers(channelId, channelType, subscribers)
	cmd := NewCMD(CMDRemoveSubscribers, data)
	cmdData, err := cmd.Marshal()
	if err != nil {
		return err
	}
	slotId := s.opts.GetSlotId(channelId)
	_, err = s.opts.Cluster.ProposeDataToSlot(s.ctx, slotId, cmdData)
	return err
}

func (s *Store) RemoveAllSubscriber(channelId string, channelType uint8) error {
	data := EncodeChannel(channelId, channelType)
	cmd := NewCMD(CMDRemoveAllSubscriber, data)
	cmdData, err := cmd.Marshal()
	if err != nil {
		return err
	}
	slotId := s.opts.GetSlotId(channelId)
	_, err = s.opts.Cluster.ProposeDataToSlot(s.ctx, slotId, cmdData)
	return err
}

func (s *Store) GetSubscribers(channelID string, channelType uint8) ([]string, error) {
	return s.wdb.GetSubscribers(channelID, channelType)
}

// AddOrUpdateChannel add or update channel
func (s *Store) AddOrUpdateChannel(channelInfo wkdb.ChannelInfo) error {
	data, err := EncodeAddOrUpdateChannel(channelInfo)
	if err != nil {
		return err
	}
	cmd := NewCMD(CMDAddOrUpdateChannel, data)
	cmdData, err := cmd.Marshal()
	if err != nil {
		return err
	}
	slotId := s.opts.GetSlotId(channelInfo.ChannelId)
	_, err = s.opts.Cluster.ProposeDataToSlot(s.ctx, slotId, cmdData)
	return err
}

func (s *Store) DeleteChannel(channelId string, channelType uint8) error {
	data := EncodeChannel(channelId, channelType)
	cmd := NewCMD(CMDDeleteChannel, data)
	cmdData, err := cmd.Marshal()
	if err != nil {
		return err
	}
	slotId := s.opts.GetSlotId(channelId)
	_, err = s.opts.Cluster.ProposeDataToSlot(s.ctx, slotId, cmdData)
	return err
}

func (s *Store) GetChannel(channelId string, channelType uint8) (wkdb.ChannelInfo, error) {
	return s.wdb.GetChannel(channelId, channelType)
}

func (s *Store) ExistChannel(channelId string, channelType uint8) (bool, error) {
	return s.wdb.ExistChannel(channelId, channelType)
}

func (s *Store) AddDenylist(channelId string, channelType uint8, uids []string) error {
	data := EncodeSubscribers(channelId, channelType, uids)
	cmd := NewCMD(CMDAddDenylist, data)
	cmdData, err := cmd.Marshal()
	if err != nil {
		return err
	}
	slotId := s.opts.GetSlotId(channelId)
	_, err = s.opts.Cluster.ProposeDataToSlot(s.ctx, slotId, cmdData)
	return err

}

func (s *Store) GetDenylist(channelId string, channelType uint8) ([]string, error) {
	return s.wdb.GetDenylist(channelId, channelType)
}

func (s *Store) ExistDenylist(channelId string, channelType uint8, uid string) (bool, error) {

	return s.wdb.ExistDenylist(channelId, channelType, uid)
}

func (s *Store) RemoveAllDenylist(channelId string, channelType uint8) error {
	cmd := NewCMD(CMDRemoveAllDenylist, nil)
	cmdData, err := cmd.Marshal()
	if err != nil {
		return err
	}
	slotId := s.opts.GetSlotId(channelId)
	_, err = s.opts.Cluster.ProposeDataToSlot(s.ctx, slotId, cmdData)
	return err
}

func (s *Store) RemoveDenylist(channelId string, channelType uint8, uids []string) error {
	data := EncodeSubscribers(channelId, channelType, uids)
	cmd := NewCMD(CMDRemoveDenylist, data)
	cmdData, err := cmd.Marshal()
	if err != nil {
		return err
	}
	slotId := s.opts.GetSlotId(channelId)
	_, err = s.opts.Cluster.ProposeDataToSlot(s.ctx, slotId, cmdData)
	return err
}

func (s *Store) AddAllowlist(channelId string, channelType uint8, uids []string) error {
	data := EncodeSubscribers(channelId, channelType, uids)
	cmd := NewCMD(CMDAddAllowlist, data)
	cmdData, err := cmd.Marshal()
	if err != nil {
		return err
	}
	slotId := s.opts.GetSlotId(channelId)
	_, err = s.opts.Cluster.ProposeDataToSlot(s.ctx, slotId, cmdData)
	return err
}

func (s *Store) GetAllowlist(channelID string, channelType uint8) ([]string, error) {
	return s.wdb.GetAllowlist(channelID, channelType)
}

func (s *Store) ExistAllowlist(channelId string, channelType uint8, uid string) (bool, error) {
	return s.wdb.ExistAllowlist(channelId, channelType, uid)
}

func (s *Store) RemoveAllAllowlist(channelId string, channelType uint8) error {
	cmdData := EncodeChannel(channelId, channelType)
	cmd := NewCMD(CMDRemoveAllAllowlist, cmdData)
	cmdData, err := cmd.Marshal()
	if err != nil {
		return err
	}
	slotId := s.opts.GetSlotId(channelId)
	_, err = s.opts.Cluster.ProposeDataToSlot(s.ctx, slotId, cmdData)
	return err
}

func (s *Store) RemoveAllowlist(channelId string, channelType uint8, uids []string) error {
	data := EncodeSubscribers(channelId, channelType, uids)
	cmd := NewCMD(CMDRemoveAllowlist, data)
	cmdData, err := cmd.Marshal()
	if err != nil {
		return err
	}
	slotId := s.opts.GetSlotId(channelId)
	_, err = s.opts.Cluster.ProposeDataToSlot(s.ctx, slotId, cmdData)
	return err
}

// 是否存在白名单
func (s *Store) HasAllowlist(channelId string, channelType uint8) (bool, error) {
	return s.wdb.HasAllowlist(channelId, channelType)
}

// func (s *Store) DeleteChannelClusterConfig(channelID string, channelType uint8) error {
// 	cmd := NewCMD(CMDChannelClusterConfigDelete, nil)
// 	cmdData, err := cmd.Marshal()
// 	if err != nil {
// 		return err
// 	}
// 	_, err = s.opts.Cluster.ProposeChannelMeta(s.ctx, channelID, channelType, cmdData)
// 	return err
// }
