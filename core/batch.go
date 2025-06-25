package core

import "iter"

type Batch interface {
	Messages() iter.Seq2[int, Message]
	Append(msg Message)
}
