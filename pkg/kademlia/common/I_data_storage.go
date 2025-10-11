package common

import (
	"github.com/linoss-7/D7024E-Project/pkg/utils"
)

type IDataStorage interface {
	Store(value DataObject) (*utils.BitArray, error)
	StoreInNetwork(value string) (*utils.BitArray, error)
	FindValueInNetwork(key *utils.BitArray) (string, []NodeInfo, error)
	FindValue(key *utils.BitArray) (string, error)
}
