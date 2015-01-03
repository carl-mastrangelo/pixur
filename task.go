package pixur

type Task interface {
	Reset()
	Run() error
}
