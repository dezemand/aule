package eventhandler

type memoryBroker struct {
	events chan Event[any]
}

type eventRouter struct {
	routes *routerNode
}

type routerNode struct {
	routes   map[string]*routerNode
	handlers []eventHandler
}

type eventHandler struct {
	f Handler[any]
}

func newEventRouter() *eventRouter {
	return &eventRouter{
		routes: &routerNode{},
	}
}

func getRoute(node *routerNode, parts []string) *routerNode {
	route := node
	for _, part := range parts {
		if route.routes == nil {
			route.routes = make(map[string]*routerNode)
		}
		nextRoute, exists := route.routes[part]
		if !exists {
			nextRoute = &routerNode{}
			route.routes[part] = nextRoute
		}
		route = nextRoute
	}
	return route
}

func registerRoute[T any](router *eventRouter, topic Topic[T], handler Handler[T]) {
	node := getRoute(router.routes, topic.Parts())
	node.handlers = append(node.handlers, eventHandler{f: func(e Event[any]) error {
		ev, ok := e.(Event[T])
		if !ok {
			return nil
		}
		return handler(ev)
	}})
}
