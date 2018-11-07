package types

import "time"

const (
	//ListURL is to list remote dir
	ListURL = "/p2pftp/list/1.0"
	//GetURL gets remote file
	GetURL = "/p2pftp/get/1.0"
	//PutURL puts local file to remote
	PutURL = "/p2pftp/put/1.0"
	//DeleteURL delete remote files
	DeleteURL = "/p2pftp/delete/1.0"
)

// ReadTimeout is to control the wait time for p2p read
const ReadTimeout = time.Hour
