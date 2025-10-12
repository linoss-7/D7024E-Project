package common

import "github.com/linoss-7/D7024E-Project/pkg/utils"

type IDataRefresher interface {
	StartRefresh(key *utils.BitArray, ticker utils.Ticker) error
	StopRefresh(key *utils.BitArray) error
}
