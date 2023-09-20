package processor

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/godeamon/chain-xevidence/config"
	"github.com/godeamon/chain-xevidence/db"
	"github.com/xuperchain/xuper-sdk-go/v2/account"
	"github.com/xuperchain/xuper-sdk-go/v2/xuper"
	"github.com/xuperchain/xuperchain/service/pb"
)

type XChainV2Processor struct {
	cfg        *config.Config
	client     *xuper.XClient
	headerChan chan *pb.InternalBlock
	db         *db.DB
	account    *account.Account
	stop       chan struct{}
}

func NewXChainV2Processor(cfg *config.Config, headerChan chan *pb.InternalBlock, db *db.DB, cfgPath string) (*XChainV2Processor, error) {
	c, err := xuper.New(cfg.MainChain.URL, xuper.WithConfigFile(cfgPath))
	if err != nil {
		return nil, err
	}
	processor := &XChainV2Processor{
		cfg:        cfg,
		client:     c,
		headerChan: headerChan,
		db:         db,
		stop:       make(chan struct{}),
	}
	return processor, nil
}

func (p *XChainV2Processor) Start() error {
	p.run()
	return nil
}

func (p *XChainV2Processor) run() {
	for {
		select {
		case <-p.stop:
			fmt.Println("Processor exit")
			return

		case header := <-p.headerChan:
			// 处理区块时，时间保证在3s内完成，如果出块时间为3s，这里每个区块处理超过3s，会导致积压很多数据。
			fmt.Println("Processor recv new block, height:", header.Height)
			err := p.process(header)
			if err != nil {
				fmt.Println("XChainV2Processor process block failed", err)
			}
			// 更新数据库已经处理过的最新区块
			err = p.db.SetLatestHeight(p.cfg.SideChain.Name, header.Height)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (p *XChainV2Processor) process(header *pb.InternalBlock) error {
	args := p.makeArgs(header)
	req, err := xuper.NewInvokeContractRequest(p.account, xuper.Xkernel3Module, "$XEvidence", "Save", args)
	if err != nil {
		return err
	}
	tx, err := p.client.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "evidence already exists") {
			return nil
		}
		return err
	}
	fmt.Println("Post txHash:", hex.EncodeToString(tx.Tx.Txid), "Args:", args)
	return nil
}

func (p *XChainV2Processor) makeArgs(header *pb.InternalBlock) map[string]string {
	type content struct {
		ChainName string `json:"chainName"`
		Height    int64  `json:"height"`
	}
	c := &content{
		ChainName: p.cfg.SideChain.Name,
		Height:    header.Height,
	}
	value, _ := json.Marshal(c)

	return map[string]string{
		"hash":    hex.EncodeToString(header.Blockid),
		"content": string(value),
	}
}

func (p *XChainV2Processor) Stop() error {
	p.stop <- struct{}{}
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}
