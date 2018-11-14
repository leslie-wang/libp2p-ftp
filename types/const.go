package types

import "time"

const (
	//PingURL is to ping remote
	PingURL = "/p2pftp/v1/ping"
	//ListURL is to list remote dir
	ListURL = "/p2pftp/v1/list"
	//GetURL gets remote file
	GetURL = "/p2pftp/v1/get"
	//PutURL puts local file to remote
	PutURL = "/p2pftp/v1/put"
	//DeleteURL delete remote files
	DeleteURL = "/p2pftp/v1/delete"
)

// ReadTimeout is to control the wait time for p2p read
const ReadTimeout = time.Hour

const (
	//QueryKeySource is the key for source
    QueryKeySource = "src"
	//QueryKeyDestination is the key for destination
	QueryKeyDestination = "dst"
)
