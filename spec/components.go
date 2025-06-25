package spec

type Component interface {
    Init(ctx ComponentContext) error
    Close(ctx ComponentContext) error
}

type Input interface {
    Component
    Read(ctx ComponentContext) (Batch, ProcessedCallback, error)
}

type Output interface {
    Component
    Write(ctx ComponentContext, batch Batch) error
}
