package config

// 目前只支持 XuperChain 架构的侧链和主链
type Config struct {
	MainChain MainChain `yaml:"mainChain"`
	SideChain SideChain `yaml:"sideChain"`
}

type MainChain struct {
	Name            string              `yaml:"name"`
	URL             string              `yaml:"url"`
	HeightInterval  int                 `yaml:"heightInterval"`  // 默认1,即每个区块都存证一次
	SenderToAccount map[string]*Account `yaml:"senderToAccount"` // key为sender地址，value 为 Account
}

type SideChain struct {
	Name string `yaml:"name"` // 名字应该是唯一的，此名字为链网络业务名字，例如 xasset
	URL  string `yaml:"url"`

	StartHeight        int64 `yaml:"startHeight"`        // 存证开始高度
	SafeHeightInterval int64 `yaml:"safeHeightInterval"` // 存证的高度和当前侧链最新高度差，可以理解为侧链不会回滚高度

	XChainVerison int    `yaml:"xChainVerison"` // 2：xchainV2版本，5：xchainV5版本，目前只支持xchainV5版本
	ChainName     string `yaml:"chainName"`     // XuperChain 平行链名字，默认 xuper

	NodeToSender map[string]string `yaml:"nodeToSender"` // key为节点矿工地址，value发送存证交易地址即MainChain中的SenderToAccount的key
}

// 目前只支持使用助记词
type Account struct {
	AccountMnemonic         string `yaml:"accountMnemonic"`         // 账户助记词，和 AccountPath 二选一，优先使用 AccountMnemonic
	AccountMnemonicLanguage int    `yaml:"accountMnemonicLanguage"` // 1中文，2英文
}

func DefaultConfig() *Config {
	return &Config{
		MainChain: MainChain{
			Name:           "xuperos",
			HeightInterval: 1,
		},
		SideChain: SideChain{
			Name:               "default_side_chain",
			SafeHeightInterval: 9,
		},
	}
}
