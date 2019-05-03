// 生成并发安全的 twitter snowflake ID

package wuuid

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	timeLayout = "2006-01-02 15:04:05"
)

type NodeSetting struct {
	BeginTimeStr   string
	NodeIDBits     uint8
	SequenceIDBits uint8
	NodeID         int64
}

// 存储基础信息的 Node 结构
type Node struct {
	lock           sync.Mutex // 保证并发安全
	timestamp      int64
	nodeID         int64
	sequenceID     int64
	beginTime      int64
	nodeIDBits     uint8
	sequenceIDBits uint8
}

// 生成一个新的 Node 类型变量
func NewNode(s NodeSetting) (*Node, error) {
	if s.NodeID < 0 || s.NodeID > (1<<s.NodeIDBits-1) {
		logrus.Error("nodeid exceeded the max value")
		return nil, fmt.Errorf("nodeid exceeded the max value")
	}

	res := &Node{
		lock:           sync.Mutex{},
		nodeID:         s.NodeID,
		nodeIDBits:     s.NodeIDBits,
		sequenceIDBits: s.SequenceIDBits,
	}

	begintime, err := time.Parse(timeLayout, s.BeginTimeStr)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	res.beginTime = begintime.UnixNano()
	return res, nil
}

func (n *Node) NextID() int64 {
	n.lock.Lock()
	defer n.lock.Unlock()

	now := time.Now().UnixNano() / 1e6 //ms

	if n.timestamp == now {
		n.sequenceID++
		if n.sequenceID > (1<<n.sequenceIDBits - 1) {
			for now == n.timestamp {
				now = time.Now().UnixNano() / 1e6
			}
			n.timestamp = now
			n.sequenceID = 0
		}
	} else {
		n.timestamp = now
		n.sequenceID = 0
	}
	return ((now - n.beginTime) << (n.nodeIDBits + n.sequenceIDBits)) | (n.nodeID << n.sequenceIDBits) | (n.sequenceID)
}
