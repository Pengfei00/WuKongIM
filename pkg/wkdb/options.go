package wkdb

type Options struct {
	NodeId            uint64
	DataDir           string
	ConversationLimit int // 最近会话查询数量限制
	SlotCount         int // 槽位数量
	// 耗时配置开启
	EnableCost   bool
	ShardNum     int               // 数据库分区数量，一但设置就不能修改
	IsCmdChannel func(string) bool // 是否是cmd频道
}

func NewOptions(opt ...Option) *Options {
	o := &Options{
		DataDir:           "./data",
		ConversationLimit: 10000,
		SlotCount:         128,
		EnableCost:        true,
		ShardNum:          16,
	}
	for _, f := range opt {
		f(o)
	}
	return o
}

type Option func(*Options)

func WithDir(dir string) Option {
	return func(o *Options) {
		o.DataDir = dir
	}
}

func WithNodeId(nodeId uint64) Option {
	return func(o *Options) {
		o.NodeId = nodeId
	}
}

func WithSlotCount(slotCount int) Option {
	return func(o *Options) {
		o.SlotCount = slotCount
	}
}

func WithConversationLimit(limit int) Option {
	return func(o *Options) {
		o.ConversationLimit = limit
	}
}

func WithEnableCost() Option {
	return func(o *Options) {
		o.EnableCost = true
	}
}

func WithShardNum(shardNum int) Option {
	return func(o *Options) {
		o.ShardNum = shardNum
	}
}

func WithIsCmdChannel(f func(string) bool) Option {
	return func(o *Options) {
		o.IsCmdChannel = f
	}
}
