package eventhandler

import (
	"strings"
)

type Topic[T any] []string

func NewTopic[T any](topic string) Topic[T] {
	parts := strings.Split(topic, ".")
	return Topic[T](parts)
}

func (t Topic[T]) String() string {
	return strings.Join([]string(t), ".")
}

func (t Topic[T]) Matches(other Topic[T]) bool {
	if len(t) != len(other) {
		return false
	}
	for i := range t {
		if t[i] != other[i] && t[i] != "*" && other[i] != "*" {
			return false
		}
	}
	return true
}

func (t Topic[T]) WithPart(part string) Topic[T] {
	return append(t, part)
}

func (t Topic[T]) Parts() []string {
	return []string(t)
}
