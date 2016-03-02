package planner

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	u "github.com/araddon/gou"
	"github.com/lytics/metafora"

	"github.com/araddon/qlbridge/datasource"
)

var (
	loggingOnce sync.Once

	// BuiltIn Default Conf, used for testing but real runtime swaps this out
	//  for a real config
	GridConf = &Conf{
		GridName:    "dataux",
		EtcdServers: strings.Split("http://127.0.0.1:2379", ","),
		NatsServers: strings.Split("nats://127.0.0.1:4222", ","),
	}
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

}

func setupLogging() {
	metafora.SetLogger(u.GetLogger()) // Configure metafora's logger
	metafora.SetLogLevel(metafora.LogLevelWarn)
	u.DiscardStandardLogger() // Discard non-sanctioned spammers
}

func RunWorkerNodes(nodeCt int, sc *datasource.RuntimeSchema) {

	loggingOnce.Do(setupLogging)

	for i := 0; i < nodeCt; i++ {
		go func(nodeId int) {
			s := serverStart(nodeCt, NodeName(uint64(nodeId)), sc)
			s.RunWorker() // blocking
		}(i)
	}
	time.Sleep(time.Millisecond * 80)
}

func NewServerGrid(nodeCt int, sc *datasource.RuntimeSchema) *Server {
	serverId, _ := NextId()
	return serverStart(nodeCt, NodeName(serverId), sc)
}

func serverStart(nodeCt int, nodeName string, sc *datasource.RuntimeSchema) *Server {
	conf := GridConf.Clone()
	conf.NodeCt = nodeCt
	conf.Hostname = nodeName
	s := &Server{Conf: conf, schemaconf: sc}
	return s
}

func NodeName(id uint64) string {
	hostname, err := os.Hostname()
	if err != nil {
		u.Errorf("error: failed to discover hostname: %v", err)
	}
	return fmt.Sprintf("%s-%d", hostname, id)
}