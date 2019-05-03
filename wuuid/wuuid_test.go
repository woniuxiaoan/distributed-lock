package wuuid

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestSnowFlake(t *testing.T) {
	node, err := NewNode(NodeSetting{
		BeginTimeStr:   "2016-01-01 01:01:01",
		NodeID:         4,
		NodeIDBits:     19,
		SequenceIDBits: 15,
	})

	if err != nil {
		logrus.Error(err)
		return
	}

	for i := 0; i <= 1500; i++ {
		fmt.Println(node.NextID())
	}
}
