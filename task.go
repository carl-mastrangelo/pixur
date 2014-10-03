package pixur

type Task interface {
	Reset()
	Run() TaskError
}

type TaskError interface {
	error
}
