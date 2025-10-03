package common

import "time"

type DataObject struct {
	Data           string
	ExpirationDate time.Time
}
