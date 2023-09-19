package config

// 目前只支持 XuperChain 架构的侧链和主链
type Config struct {
	MainChain MainChain `yaml:"mainChain"`
	SideChain SideChain `yaml:"sideChain"`
}

type MainChain struct {
	Name           string `yaml:"name"`
	URL            string `yaml:"url"`
	HeightInterval int    `yaml:"heightInterval"` // 默认1,即每个区块都存证一次

	AccountMnemonic         string `yaml:"accountMnemonic"`         // 账户助记词，和 AccountPath 二选一，优先使用 AccountMnemonic
	AccountMnemonicLanguage int    `yaml:"accountMnemonicLanguage"` // 1中文，2英文

	AccountPath   string `yaml:"accountPath"`   // 账户路径
	AccountPasswd string `yaml:"accountPasswd"` // AccountPath 是加密过的则此处需要配置密码
}

type SideChain struct {
	Name string `yaml:"name"` // 名字应该是唯一的，此名字为链网络业务名字，例如 xasset
	URL  string `yaml:"url"`

	StartHeight        int64 `yaml:"startHeight"`        // 存证开始高度
	SafeHeightInterval int64 `yaml:"safeHeightInterval"` // 存证的高度和当前侧链最新高度差，可以理解为侧链不会回滚高度

	XChainVerison int    `yaml:"xChainVerison"` // 2：xchainV2版本，5：xchainV5版本，目前只支持这两个配置，默认xchainV5版本
	ChainName     string `yaml:"chainName"`     // XuperChain 平行链名字，默认 xuper
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
