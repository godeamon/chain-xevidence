package listener

import (
	"fmt"
	"time"

	"github.com/godeamon/chain-xevidence/config"
	"github.com/godeamon/chain-xevidence/log"
	"github.com/xuperchain/xuper-sdk-go/v2/xuper"
	"github.com/xuperchain/xuperchain/service/pb"
)

// XChainListener 侧链为 XuperChain 的监听器
type XChainListener struct {
	ChainName string
	cfg       *config.Config

	headerChan chan *pb.InternalBlock // type 可以有 buffer，当程序重启后，从数据库获取 latestHeight，不会导致数据丢失

	client       *xuper.XClient
	latestHeight int64

	stop chan struct{}
}

func NewXChainListener(cfg *config.Config, headerChan chan *pb.InternalBlock, latestHeight int64, cfgPath string) (*XChainListener, error) {
	log.Log.Info("XuperChain Listner New")
	c, err := xuper.New(cfg.SideChain.URL, xuper.WithConfigFile(cfgPath))
	if err != nil {
		return nil, err
	}
	return &XChainListener{
		cfg:          cfg,
		headerChan:   headerChan,
		client:       c,
		latestHeight: latestHeight,
		stop:         make(chan struct{}),
	}, nil
}

// Start called by async goroutinue
func (x *XChainListener) Start() error {
	log.Log.Info("XuperChain Listner Start")
	x.run()
	return nil
}

func (x *XChainListener) run() {
	for {
		select {
		case <-x.stop:
			fmt.Println("XuperChain listener exit")
			log.Log.Info("XuperChain listner exit")
			return
		default:
			tip := x.getTipBlockHeight()
			blocks := x.getSafeBlocks(tip)
			if len(blocks) == 0 {
				time.Sleep(time.Second * 1) // 暂定1s查询一次
				continue
			}
			for _, b := range blocks {
				// 这里是同步的，不能使用异步，避免 processor 处理时高度错乱
				select {
				case <-x.stop:
					fmt.Println("XuperChain listener exit")
					log.Log.Info("XuperChain listner exit")
					return
				case x.headerChan <- b:
					x.latestHeight = b.Height
				}
			}
			// 这里不需要 sleep，因为 processor 处理还需要一段时间
		}
	}
}

func (x *XChainListener) getTipBlockHeight() int64 {
	status, err := x.client.QuerySystemStatus()
	if err != nil {
		fmt.Println("XChainListener QuerySystemStatus failed:", err)
	}
	for _, s := range status.SystemsStatus.BcsStatus {
		if s.Bcname == x.cfg.SideChain.ChainName {
			h := s.GetBlock().Height
			log.Log.Info("XuperChain listner side chain status", "height", h)
			return h
		}
	}
	panic("xuper chain listener getTipBlockHeight failed, invalid side chain name:" + x.cfg.SideChain.ChainName)
}

func (x *XChainListener) getSafeBlocks(tipHeight int64) []*pb.InternalBlock {
	blocks := make([]*pb.InternalBlock, 0)
	diff := tipHeight - x.cfg.SideChain.SafeHeightInterval - x.latestHeight
	if diff <= 0 {
		return blocks
	}
	to := tipHeight - x.cfg.SideChain.SafeHeightInterval
	if diff > 128 {
		// 如果间隔太大，一次最多256个区块，避免内存占用太多
		// 后期可以放到配置文件
		to = x.latestHeight + 128
	}
	blocks, err := x.queryBlocks(x.latestHeight+1, to)
	if err != nil {
		fmt.Println("XChainListener queryBlocks failed:", err)
		log.Log.Error("XuperChain listner queryBlocks failed", "err", err)
	}
	return blocks
}

func (x *XChainListener) queryBlocks(from, to int64) ([]*pb.InternalBlock, error) {
	blocks := make([]*pb.InternalBlock, 0)
	for i := from; i <= to; i++ {
		block, err := x.client.QueryBlockByHeight(i)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block.GetBlock())
	}
	return blocks, nil

}

func (x *XChainListener) Stop() error {
	x.stop <- struct{}{}
	if x.client != nil {
		return x.client.Close()
	}
	return nil
}
