package worker

import (
	"fmt"

	"github.com/godeamon/chain-xevidence/config"
	"github.com/godeamon/chain-xevidence/db"
	"github.com/godeamon/chain-xevidence/worker/listener"
	"github.com/godeamon/chain-xevidence/worker/processor"
	"github.com/xuperchain/xuperchain/service/pb"
)

// Manager 统一管理所有 worker，根据配置文件，创建维护不同的 worker。
type Manager struct {
	cfg     *config.Config
	workers []*Worker
	db      *db.DB
}

type Worker struct {
	listener  listener.Listener
	processor processor.Processor
}

func NewManager(cfg *config.Config, cfgPath string) (*Manager, error) {
	m := &Manager{
		cfg: cfg,
	}
	db := db.New()
	m.db = db
	headerChan := make(chan *pb.InternalBlock)
	latestHeight := m.getLatestHeightFromDB()

	l, err := m.newListener(cfg, headerChan, latestHeight, cfgPath)
	if err != nil {
		return nil, err
	}

	p, err := m.newProcessor(cfg, headerChan, db, cfgPath)
	if err != nil {
		return nil, err
	}

	w := &Worker{
		listener:  l,
		processor: p,
	}
	m.workers = append(m.workers, w) // 后期可以根据配置文件创建更多的 worker
	return m, nil
}

func (m *Manager) newListener(cfg *config.Config, headerChan chan *pb.InternalBlock, latestHeight int64, cfgPath string) (listener.Listener, error) {
	switch cfg.SideChain.XChainVerison {
	case 2:
		panic("xchain version 2 not support yet")
	case 5:
		l, err := listener.NewXChainListener(cfg, headerChan, latestHeight, cfgPath)
		if err != nil {
			return nil, err
		}
		return l, nil
	default:
		return nil, fmt.Errorf("invalid xChainVersion: %d", cfg.SideChain.XChainVerison)
	}
}

func (m *Manager) newProcessor(cfg *config.Config, headerChan chan *pb.InternalBlock, db *db.DB, cfgPath string) (processor.Processor, error) {
	switch cfg.SideChain.XChainVerison {
	case 2:
		panic("xchain version 2 not support yet")
	case 5:
		fmt.Println("xchain v5")
		l, err := processor.NewXChainProcessor(cfg, headerChan, db, cfgPath)
		if err != nil {
			return nil, err
		}
		return l, nil
	default:
		return nil, fmt.Errorf("invalid xChainVersion: %d", cfg.SideChain.XChainVerison)
	}
}

func (m *Manager) getLatestHeightFromDB() int64 {
	height, err := m.db.GetLatestHeight(m.cfg.SideChain.Name)
	if err != nil {
		return m.cfg.SideChain.StartHeight
	}
	return height
}

func (m *Manager) Start() {
	for _, w := range m.workers {
		w.Run()
	}
}

func (m *Manager) Stop() {
	fmt.Println("Manager Stop")
	for _, w := range m.workers {
		w.Stop()
	}
}

func NewWorker() *Worker {
	w := &Worker{}
	return w
}

func (w *Worker) Run() error {
	go w.processor.Start()
	go w.listener.Start()
	return nil
}

func (w *Worker) Stop() error {
	w.listener.Stop()
	w.processor.Stop()
	return nil
}
