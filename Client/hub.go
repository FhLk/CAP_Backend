package Client

// Hub maintains the set of active clients and broadcasts messages to the
type Hub struct {
	// put registered clients into the room.
	rooms map[string]map[*connection]bool
	// Inbound messages from the clients.
	broadcast chan message

	// Register requests from the clients.
	register chan subscription

	// Unregister requests from clients.
	unregister chan subscription
}

type message struct {
	Room string `json:"room"`
	Data []byte `json:"data"`
}

var H = &Hub{
	broadcast:  make(chan message),
	register:   make(chan subscription),
	unregister: make(chan subscription),
	rooms:      make(map[string]map[*connection]bool),
}

func (h *Hub) Run() {
	for {
		select {
		case s := <-h.register:
			connections := h.rooms[s.room]
			if connections == nil {
				connections = make(map[*connection]bool)
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
