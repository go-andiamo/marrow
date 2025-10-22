package marrow

type Runnable interface {
	Run(ctx Context) error
	Framed
}
