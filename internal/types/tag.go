package types

import "time"

type Tag struct {
	ID   int64
	Name string
}

type ContentAutoTag struct {
	ID        int64
	TagID     int64
	URLRegex  string
	CreatedAt time.Time
}

type PageAutoTag struct {
	ID        int64
	TagID     int64
	PathRegex string
	CreatedAt time.Time
}
