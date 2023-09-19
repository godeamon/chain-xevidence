package processor

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/godeamon/chain-xevidence/config"
	"github.com/godeamon/chain-xevidence/db"
	"github.com/xuperchain/xuper-sdk-go/v2/account"
	"github.com/xuperchain/xuper-sdk-go/v2/xuper"
	"github.com/xuperchain/xuperchain/service/pb"
)

type XChainProcessor struct {
	cfg        *config.Config
	client     *xuper.XClient
	headerChan chan *pb.InternalBlock
	db         *db.DB
	account    *account.Account
	stop       chan struct{}

	txWithRequest chan *TxWithRequest
}

type TxWithRequest struct {
	Tx  *pb.Transaction
	Req *xuper.Request
}

func NewXChainProcessor(cfg *config.Config, headerChan chan *pb.InternalBlock, db *db.DB, account *account.Account, cfgPath string) (*XChainProcessor, error) {
	c, err := xuper.New(cfg.MainChain.URL, xuper.WithConfigFile(cfgPath))
	if err != nil {
		return nil, err
	}
	processor := &XChainProcessor{
		cfg:           cfg,
		client:        c,
		headerChan:    headerChan,
		db:            db,
		account:       account,
		stop:          make(chan struct{}),
		txWithRequest: make(chan *TxWithRequest, 10), // 暂定缓存10个
	}
	return processor, nil
}

func (p *XChainProcessor) Start() error {
	go p.checkAndResendFailedTx()
	p.run()
	return nil
}

func (p *XChainProcessor) run() {
	/*
		1、for 循环从 channel 中读取区块头
		2、根据配置以及获取到的区块头，打包 xchain 存证交易
			2.1、配置文件中配置了交易频率
			2.2、可以等待一定时间确认交易成功上链
		3、交易成功后，更新本地处理过的最新区块高度
	*/
	count := 0
	for {
		select {
		case <-p.stop:
			fmt.Println("Processor exit")
			return

		case header := <-p.headerChan:
			// 处理区块时，时间保证在3s内完成，如果出块时间为3s，这里每个区块处理超过3s，会导致积压很多数据。
			fmt.Println("Processor recv new block, height:", header.Height)
			count++
			if count >= p.cfg.MainChain.HeightInterval {
				err := p.process(header)
				if err != nil {
					panic(err)
				}
				count = 0 // 重置计数

				// 更新数据库已经处理过的最新区块
				err = p.db.SetLatestHeight(p.cfg.SideChain.Name, header.Height)
				if err != nil {
					panic(err)
				}
			}
		}
	}
}

// 检查以及发送的交易是否成功上链，如果没有，则重新发送
// 如果失败的较多，重新发送占用时间较长，应同步进行，所有交易重新上链后再释放锁
func (p *XChainProcessor) checkAndResendFailedTx() {
	// TODO
	// 1、根据交易ID查询交易，如果交易中包含blockID信息，则确认为上链成功，如果不包含则认为还没上链
	// 2、如果交易未上链，则等待0.5s再次查询，超过10s一直未上链则标记为上链失败
	// 3、上链失败的交易入库，创建新的线程重新发起交易
	// 4、重新发起的交易如果依然失败则丢弃交易，不再重试
	for {
		select {
		case <-p.stop:
			fmt.Println("Processor exit")
			return

		case txr := <-p.txWithRequest:
			fmt.Println(txr.Req) // todo
		}
	}
}

func (p *XChainProcessor) process(header *pb.InternalBlock) error {
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
	time.Sleep(time.Second / 2) // 等待0.5s再返回
	// fmt.Println(tx)             //todo wait tx
	fmt.Println("Post txHash:", hex.EncodeToString(tx.Tx.Txid), "Args:", args)
	// todo 这里把tx异步给到 checkAndResendFailedTx 线程来保证交易上链成功
	select {
	case p.txWithRequest <- &TxWithRequest{
		Tx:  tx.Tx,
		Req: req,
	}:
	default:
		go func() {
			p.txWithRequest <- &TxWithRequest{
				Tx:  tx.Tx,
				Req: req,
			}
		}()
	}

	return nil
}

func (p *XChainProcessor) makeArgs(header *pb.InternalBlock) map[string]string {
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

func (p *XChainProcessor) Stop() error {
	p.stop <- struct{}{}
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}
