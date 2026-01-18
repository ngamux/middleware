package requestid

import "github.com/google/uuid"

type config struct {
	KeyHeader func() string
	ID        func() string
	OnError   func(error)
}

func cKeyHeader() string {
	return "X-Request-ID"
}

func cID() string {
	_id, _ := uuid.NewV7()
	return _id.String()
}

func WithKeyHeader(key func() string) func(*config) {
	return func(c *config) {
		if key == nil {
			key = cKeyHeader
		}

		c.KeyHeader = key
	}
}

func WithID(id func() string) func(*config) {
	return func(c *config) {
		if id == nil {
			id = cID
		}

		c.ID = id
	}
}
