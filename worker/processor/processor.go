package processor

type Processor interface {
	Start() error
	Stop() error
}
