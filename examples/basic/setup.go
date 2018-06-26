package basic

import (
	"fmt"
	"time"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/examples/basic/messages"
	"github.com/perlin-network/noise/grpc_utils"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/network/discovery"
)

type ClusterNode struct {
	Host             string
	Port             int
	Peers            []string
	Net              *network.Network
	BufferedMessages []*messages.BasicMessage
}

func (c *ClusterNode) Handle(client *network.PeerClient, raw *network.IncomingMessage) error {
	message := raw.Message.(*messages.BasicMessage)

	if c.BufferedMessages == nil {
		c.BufferedMessages = []*messages.BasicMessage{}
	}
	c.BufferedMessages = append(c.BufferedMessages, message)

	return nil
}

func (c *ClusterNode) PopMessage() *messages.BasicMessage {
	if len(c.BufferedMessages) == 0 {
		return nil
	}
	var retVal *messages.BasicMessage
	retVal, c.BufferedMessages = c.BufferedMessages[0], c.BufferedMessages[1:]
	return retVal
}

var blockTimeout = 10 * time.Second

// SetupCluster - Sets up a connected group of nodes in a cluster
func SetupCluster(nodes []*ClusterNode) error {
	peers := []string{}

	for i := 0; i < len(nodes); i++ {
		node := nodes[i]
		keys := crypto.RandomKeyPair()
		peers = append(peers, fmt.Sprintf("%s:%d", node.Host, node.Port))

		builder := &builders.NetworkBuilder{}
		builder.SetKeys(keys)
		builder.SetHost(node.Host)
		builder.SetPort(node.Port)

		discovery.BootstrapPeerDiscovery(builder)

		builder.AddProcessor((*messages.BasicMessage)(nil), nodes[i])

		net, err := builder.BuildNetwork()
		if err != nil {
			return err
		}
		node.Net = net

		go net.Listen()
	}

	for i := 0; i < len(nodes); i++ {
		if err := grpc_utils.BlockUntilConnectionReady(nodes[i].Host, nodes[i].Port, blockTimeout); err != nil {
			return fmt.Errorf("Error: port was not available, cannot bootstrap node %d peers, err=%+v", i, err)
		}
	}

	for _, node := range nodes {
		node.Net.Bootstrap(node.Peers...)

		// TODO: seems there's another race condition with Bootstrap, use a sleep for now
		time.Sleep(1 * time.Second)
	}

	return nil
}
