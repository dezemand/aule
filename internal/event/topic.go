package event

import "strings"

const TopicDelimitor = "."

type Topic[T any] []string

func NewTopic[T any](topic string) Topic[T] {
	parts := strings.Split(topic, TopicDelimitor)
	return Topic[T](parts)
}

func (t Topic[T]) String() string {
	return strings.Join(t, TopicDelimitor)
}

func (t Topic[T]) Parts() []string {
	return []string(t)
}

func (t Topic[T]) Equals(other Topic[T]) bool {
	if len(t) != len(other) {
		return false
	}
	for i := range t {
		if t[i] != other[i] {
			return false
		}
	}
	return true
}

func (t Topic[T]) Event(payload T, options ...EventOption) Event[T] {
	return NewEvent(t, payload, options...)
}
