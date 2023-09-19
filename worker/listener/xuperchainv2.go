package listener

import (
	"fmt"
	"time"

	"github.com/godeamon/chain-xevidence/config"
	"github.com/xuperchain/xuper-sdk-go/v2/xuper"
	"github.com/xuperchain/xuperchain/service/pb"
)

// XChainV2Listener 兼容 XuperChain v2 版本接口的 listener
type XChainV2Listener struct {
	ChainName string
	cfg       *config.Config

	headerChan chan *pb.InternalBlock // type 可以有 buffer，当程序重启后，从数据库获取 latestHeight，不会导致数据丢失

	client       *xuper.XClient
	latestHeight int64

	stop chan struct{}
}

func NewXChainV2Listener(cfg *config.Config, headerChan chan *pb.InternalBlock, latestHeight int64, cfgPath string) (*XChainV2Listener, error) {
	c, err := xuper.New(cfg.SideChain.URL, xuper.WithConfigFile(cfgPath))
	if err != nil {
		return nil, err
	}
	return &XChainV2Listener{
		cfg:          cfg,
		headerChan:   headerChan,
		client:       c,
		latestHeight: latestHeight,
		stop:         make(chan struct{}),
	}, nil
}

func (x *XChainV2Listener) Start() error {
	x.run()
	return nil
}

func (x *XChainV2Listener) Stop() error {
	x.stop <- struct{}{}
	if x.client != nil {
		return x.client.Close()
	}
	return nil
}

// xchainv2 不支持通过高度查询区块，且通过区块ID查询在某些情况不兼容因此采用下面方案：
// 只能是每次查询 SystemStatus 接口查询到区块ID，只有和上一次查询的的不一致即进行存证
// 不考虑丢失情况以及区块回滚情况
func (x *XChainV2Listener) run() {
	for {
		select {
		case <-x.stop:
			fmt.Println("Listener exit")
			return
		default:
			status, err := x.client.QuerySystemStatus()
			if err != nil {
				fmt.Println("query system status err:", err)
				continue
			}
			for _, s := range status.SystemsStatus.BcsStatus {
				if s.Bcname == x.cfg.SideChain.ChainName {
					if x.latestHeight != s.Block.Height {
						x.headerChan <- s.Block
						x.latestHeight = s.Block.Height
						fmt.Println("Listener recv new block height: ", s.Block.Height)
					}
					time.Sleep(time.Second * 1)
					break
				}
			}
		}
	}
}
