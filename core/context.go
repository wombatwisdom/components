package core

import "context"

type ComponentContext interface {
    ExpressionFactory
    MessageFactory

    Context() context.Context

    Input(name string) (Input, error)
    Output(name string) (Output, error)
}
