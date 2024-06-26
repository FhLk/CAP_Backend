package Client

type Hub struct {
	rooms      map[string]map[*Connection]bool
	broadcast  chan message
	register   chan Subscription
	unregister chan Subscription
}

type message struct {
	Room string `json:"room"`
	Data []byte `json:"data"`
}

var H = &Hub{
	broadcast:  make(chan message),
	register:   make(chan Subscription),
	unregister: make(chan Subscription),
	rooms:      make(map[string]map[*Connection]bool),
}

func (h *Hub) Run() {
	for {
		select {
		case s := <-h.register:
			connections := h.rooms[s.room]
			if connections == nil {
				connections = make(map[*Connection]bool)
				h.rooms[s.room] = connections
			}
			h.rooms[s.room][s.conn] = true
		case s := <-h.unregister:
			connections := h.rooms[s.room]
			if connections != nil {
				if _, ok := connections[s.conn]; ok {
					delete(connections, s.conn)
					close(s.conn.send)
					if len(connections) == 0 {
						delete(h.rooms, s.room)
					}
				}
			}
		case m := <-h.broadcast:
			connections := h.rooms[m.Room]
			for c := range connections {
				select {
				case c.send <- m.Data:
				default:
					close(c.send)
					delete(connections, c)
					if len(connections) == 0 {
						delete(h.rooms, m.Room)
					}
				}
			}
		}
	}
}
