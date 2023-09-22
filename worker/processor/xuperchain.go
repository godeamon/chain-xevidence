package processor

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/godeamon/chain-xevidence/config"
	"github.com/godeamon/chain-xevidence/db"
	"github.com/godeamon/chain-xevidence/log"
	"github.com/xuperchain/xuper-sdk-go/v2/account"
	"github.com/xuperchain/xuper-sdk-go/v2/xuper"
	"github.com/xuperchain/xuperchain/service/pb"
)

type XChainProcessor struct {
	cfg        *config.Config
	client     *xuper.XClient
	headerChan chan *pb.InternalBlock
	db         *db.DB
	stop       chan struct{}

	senderToAccount map[string]*account.Account
	nodeToSender    map[string]string

	needCheck chan *EvidenceBlock

	m *sync.Mutex
}

type EvidenceBlock struct {
	Height   int64
	Blockid  []byte
	Proposer []byte
}

func NewXChainProcessor(cfg *config.Config, headerChan chan *pb.InternalBlock, db *db.DB, cfgPath string) (*XChainProcessor, error) {
	log.Log.Info("XuperChain Processor New")
	c, err := xuper.New(cfg.MainChain.URL, xuper.WithConfigFile(cfgPath))
	if err != nil {
		return nil, err
	}
	if len(cfg.MainChain.SenderToAccount) == 0 {
		return nil, errors.New("XChainProcessor config sender to account is empty")
	}
	if _, ok := cfg.MainChain.SenderToAccount["default"]; !ok {
		return nil, errors.New("XChainProcessor config node to sender default is required")
	}
	senderToAccount := make(map[string]*account.Account, len(cfg.MainChain.SenderToAccount))
	for s, a := range cfg.MainChain.SenderToAccount {
		acc, err := account.RetrieveAccount(a.AccountMnemonic, a.AccountMnemonicLanguage)
		if err != nil {
			return nil, err
		}
		senderToAccount[s] = acc
	}

	processor := &XChainProcessor{
		cfg:             cfg,
		client:          c,
		headerChan:      headerChan,
		db:              db,
		stop:            make(chan struct{}),
		senderToAccount: senderToAccount,
		nodeToSender:    cfg.SideChain.NodeToSender,
		needCheck:       make(chan *EvidenceBlock, 10), // 暂定缓存10个
		m:               &sync.Mutex{},
	}
	return processor, nil
}

func (p *XChainProcessor) Start() error {
	log.Log.Info("XuperChain Processor Start")
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
			log.Log.Info("XuperChain Processor recv block", "height", header.Height)
			count++
			if count >= p.cfg.MainChain.HeightInterval {
				eblock := &EvidenceBlock{
					Height:   header.Height,
					Blockid:  header.Blockid,
					Proposer: header.Proposer,
				}
			Loop:
				select {
				case <-p.stop:
					fmt.Println("XuperChain Processor exit")
					log.Log.Info("XuperChain Processor exit")
					return
				default:
					err := p.process(eblock, true)
					if err != nil {
						fmt.Println(err)
						log.Log.Info("XuperChain Processor process block failed", "err", err)
						time.Sleep(time.Millisecond * 500)
						goto Loop
					}
					count = 0 // 重置计数

					// 更新数据库已经处理过的最新区块
					err = p.db.SetLatestHeight(p.cfg.SideChain.Name, header.Height)
					if err != nil {
						panic(err)
					}
					break Loop
				}

			}
		}
	}
}

// 检查以及发送的交易是否成功上链，如果没有，则重新发送
// 如果失败的较多，重新发送占用时间较长，应同步进行，所有交易重新上链后再释放锁
func (p *XChainProcessor) checkAndResendFailedTx() {
	for {
		select {
		case <-p.stop:
			fmt.Println("Processor exit")
			log.Log.Info("XuperChain Processor exit")
			return

		case eblock := <-p.needCheck:
			log.Log.Debug("XuperChain Processor check gotoutinue recv block", "height", eblock.Height)
			err := p.check(eblock)
			if err != nil {
				fmt.Println(err)
				log.Log.Error("XuperChain Processor check gotoutinue check block failed", "err", err)
			}
		}
	}
}

func (p *XChainProcessor) check(eblock *EvidenceBlock) error {
	err := p.setNeedCheckBlock(eblock)
	if err != nil {
		return err
	}

	h, err := p.getCheckedHeight()
	if err != nil {
		return err
	}

	if eblock.Height-h < 10 {
		return nil
	}

	var (
		needCheckBlock *EvidenceBlock
		checkedHeight  int64
	)

	for checkedHeight = int64(h + 1); checkedHeight < eblock.Height-h-10; checkedHeight++ {
		needCheckBlock, err = p.getNeedCheckBlock(checkedHeight)
		if err != nil {
			return err
		}
		if needCheckBlock != nil {
			break
		}
	}
	if needCheckBlock == nil {
		return p.setCheckedHeight(checkedHeight)
	}

	acc, err := p.getSender(needCheckBlock)
	if err != nil {
		return err
	}
	args := p.makeArgs(needCheckBlock)
	req, err := xuper.NewInvokeContractRequest(acc, xuper.Xkernel3Module, "$XEvidence", "Get", args)
	if err != nil {
		return err
	}
	_, err = p.client.PreExecTx(req)
	if err == nil {
		// 此时存证哈希以及找到，删除需要检查的block同时更新已检查的高度
		err := p.delNeedCheckBlock(needCheckBlock)
		if err != nil {
			return err
		}
		return p.setCheckedHeight(checkedHeight)
	}
	if !strings.Contains(err.Error(), "evidence not found") {
		// 此时发生错误，且不是查询存证失败，返回错误
		// 并且这里不会更新数据库
		return err
	}

	// 此时存证不存在，需要重新进行存证，这里只重试一次
	err = p.process(needCheckBlock, false)
	if err != nil {
		return err
	}
	err = p.delNeedCheckBlock(needCheckBlock)
	if err != nil {
		return err
	}
	return p.setCheckedHeight(checkedHeight)
}

func (p *XChainProcessor) process(header *EvidenceBlock, needCheck bool) error {
	// 失败交易重新执行时，避免冲突这里必须 lock！
	p.m.Lock()
	defer p.m.Unlock()

	acc, err := p.getSender(header)
	if err != nil {
		return err
	}
	args := p.makeArgs(header)
	log.Log.Debug("XuperChain Processor process", "height", header.Height, "proposer", string(header.Proposer), "sender", acc.Address, "args", args)
	req, err := xuper.NewInvokeContractRequest(acc, xuper.Xkernel3Module, "$XEvidence", "Save", args)
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
	time.Sleep(time.Millisecond * 500) // 等待0.5s再返回
	log.Log.Info("XuperChain Processor process done", "txHash", hex.EncodeToString(tx.Tx.Txid))

	if needCheck {
		select {
		case p.needCheck <- header:
		default:
			go func() {
				p.needCheck <- header
			}()
		}
	}

	return nil
}

func (p *XChainProcessor) getSender(header *EvidenceBlock) (*account.Account, error) {
	miner := string(header.Proposer)
	defaultSender, ok := p.senderToAccount["default"]
	if !ok {
		return nil, errors.New("default not exist")
	}
	sender, ok := p.nodeToSender[miner]
	if !ok {
		return defaultSender, nil
	}
	acc, ok := p.senderToAccount[sender]
	if !ok {
		return defaultSender, nil
	}
	return acc, nil
}

func (p *XChainProcessor) makeArgs(header *EvidenceBlock) map[string]string {
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

func (p *XChainProcessor) setNeedCheckBlock(block *EvidenceBlock) error {
	key := []byte(p.cfg.SideChain.Name + "_needcheck_" + strconv.FormatInt(block.Height, 10))
	value, _ := json.Marshal(block)
	return p.db.Set(key, value)
}

func (p *XChainProcessor) getNeedCheckBlock(height int64) (*EvidenceBlock, error) {
	key := []byte(p.cfg.SideChain.Name + "_needcheck_" + strconv.FormatInt(height, 10))
	value, err := p.db.Get(key)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return nil, err
	}
	if len(value) == 0 {
		return nil, nil
	}
	block := &EvidenceBlock{}
	err = json.Unmarshal(value, block)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (p *XChainProcessor) delNeedCheckBlock(block *EvidenceBlock) error {
	key := []byte(p.cfg.SideChain.Name + "_needcheck_" + strconv.FormatInt(block.Height, 10))
	return p.db.Del(key)
}

func (p *XChainProcessor) setCheckedHeight(height int64) error {
	key := []byte(p.cfg.SideChain.Name + "_checkedheight")
	value, _ := json.Marshal(height)
	return p.db.Set(key, value)
}

func (p *XChainProcessor) getCheckedHeight() (int64, error) {
	key := []byte(p.cfg.SideChain.Name + "_checkedheight")
	value, err := p.db.Get(key)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return 0, err
	}
	if len(value) == 0 {
		return 0, nil
	}
	h := int64(0)
	err = json.Unmarshal(value, &h)
	return h, err
}
