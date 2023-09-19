package listener

// Listener 监听侧链上的区块，根据不同侧链有不同实现
type Listener interface {
	Start() error
	Stop() error
}
