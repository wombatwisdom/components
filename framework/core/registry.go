package core

type Registry interface {
	RegisterInput(name string, input Input) error
	GetInput(name string) (Input, error)

	RegisterOutput(name string, output Output) error
	GetOutput(name string) (Output, error)
}
