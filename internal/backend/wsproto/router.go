package wsproto

import "context"

type Router struct {
}

func NewRouter() *Router {
	return &Router{}
}

func (r *Router) recv(ctx context.Context, message *Envelope) error {
	return nil
}
