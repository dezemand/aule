package eventhandler

type MemoryEventHandler struct {
	handlers map[string][]Handler
}

func NewMemoryEventHandler() *MemoryEventHandler {
	return &MemoryEventHandler{
		handlers: make(map[string][]Handler),
	}
}

func (h *MemoryEventHandler) Register(eventType string, handler Handler) {
	h.handlers[eventType] = append(h.handlers[eventType], handler)
}

func (h *MemoryEventHandler) Emit(event Event) error {
	handlers := h.handlers[event.Type()]
	for _, handler := range handlers {
		handler.Handle(event)
	}
	return nil
}
