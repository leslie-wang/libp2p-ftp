package node

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p-peer"

	"github.com/leslie-wang/p2pftp/handler"
	"github.com/leslie-wang/p2pftp/types"

	"github.com/pkg/errors"

	cid "github.com/ipfs/go-cid"
	iaddr "github.com/ipfs/go-ipfs-addr"
	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	mh "github.com/multiformats/go-multihash"
)

// Node is the structure for current node
type Node struct {
	host            host.Host
	rendezvousPoint cid.Cid
	kadDHT          *dht.IpfsDHT
	handler         *handler.Handler
}

// ID returns local node's ID
func (n *Node) ID() peer.ID {
	return n.host.ID()
}

// DiscoverPeers discover peers in the DHT network
func (n *Node) DiscoverPeers(ctx context.Context) ([]pstore.PeerInfo, error) {
	tctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	peers, err := n.kadDHT.FindProviders(tctx, n.rendezvousPoint)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Found %d peers!\n", len(peers))

	for _, p := range peers {
		fmt.Println("Peer: ", p)
	}

	return peers, nil
}

// StartNode starts current node and connect to dht network
func StartNode(ctx context.Context, rendezvous string, bootstrapPeers []string, announce bool) (*Node, error) {
	// libp2p.New constructs a new libp2p Host.
	// Other options can be added here.
	var err error
	node := &Node{}

	v1b := cid.V1Builder{Codec: cid.Raw, MhType: mh.SHA2_256}
	node.rendezvousPoint, err = v1b.Sum([]byte(rendezvous))
	if err != nil {
		return nil, err
	}

	node.host, err = libp2p.New(ctx)
	if err != nil {
		return nil, err
	}

	node.handler = handler.NewHandler(node.host)

	node.kadDHT, err = dht.New(ctx, node.host)
	if err != nil {
		return nil, err
	}

	// Let's connect to the bootstrap nodes first. They will tell us about the other nodes in the network.
	ok := false
	for _, peerAddr := range bootstrapPeers {
		addr, _ := iaddr.ParseString(peerAddr)
		peerinfo, _ := pstore.InfoFromP2pAddr(addr.Multiaddr())

		if err := node.host.Connect(ctx, *peerinfo); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Connection established with bootstrap node: ", *peerinfo)
			ok = true
		}
	}
	if !ok {
		return nil, errors.New("Unable to connect any bootstrap nodes")
	}

	if !announce {
		return node, nil
	}

	// register handler for each method
	node.handler.MkRoutes()

	// announce myself
	fmt.Printf("Announcing ourselves: %s (%s)\n", node.host.ID().String(), node.host.ID().Pretty())
	tctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	return node, node.kadDHT.Provide(tctx, node.rendezvousPoint, true)
}

// ListRequest sends list request to remote peer
func (n *Node) ListRequest(ctx context.Context, pid peer.ID, dir string) (files []string, err error) {
	if !path.IsAbs(dir) {
		return nil, errors.New("please use absolute path")
	}
	stream, err := n.host.NewStream(ctx, pid, types.ListURL)
	if err != nil {
		return nil, err
	}
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	if _, err := rw.WriteString(fmt.Sprintf("%s\n", dir)); err != nil {
		return nil, err
	}
	if err := rw.Flush(); err != nil {
		return nil, err
	}

	ch := make(chan []string)
	e := make(chan error)

	go func() {
		l := []string{}
		for {
			file, err := rw.ReadString('\n')
			if err != nil {
				e <- err
			}
			if file == "\n" {
				ch <- l
				break
			} else {
				l = append(l, strings.TrimSpace(file))
			}
		}
		close(ch)
		close(e)
	}()

	select {
	case files = <-ch:
		return files, nil
	case err = <-e:
		return nil, err
	case <-time.After(types.ReadTimeout):
		return nil, errors.New("Read Timeout")
	}
}

// DeleteRequest sends delete request to remote peer
func (n *Node) DeleteRequest(ctx context.Context, pid peer.ID, dir string) error {
	if !path.IsAbs(dir) {
		return errors.New("please use absolute path")
	}
	stream, err := n.host.NewStream(ctx, pid, types.DeleteURL)
	if err != nil {
		return err
	}
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	if _, err := rw.WriteString(fmt.Sprintf("%s\n", dir)); err != nil {
		return err
	}
	if err := rw.Flush(); err != nil {
		return err
	}

	e := make(chan error)

	go func() {
		for i := 0; i < 2; i++ {
			_, err := rw.ReadString('\n')
			if err != nil {
				e <- err
			}
		}
		close(e)
	}()

	select {
	case err := <-e:
		return err
	case <-time.After(types.ReadTimeout):
		return errors.New("Read Timeout")
	}
}

// GetRequest sends get request to remote peer
func (n *Node) GetRequest(ctx context.Context, pid peer.ID, filename string, dst io.Writer) error {
	if !path.IsAbs(filename) {
		return errors.New("please use absolute path")
	}
	stream, err := n.host.NewStream(ctx, pid, types.GetURL)
	if err != nil {
		return err
	}
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	if _, err := rw.WriteString(fmt.Sprintf("%s\n", filename)); err != nil {
		return err
	}
	if err := rw.Flush(); err != nil {
		return err
	}

	s := make(chan string)
	e := make(chan error)

	go func() {
		size, err := rw.ReadString('\n')
		if err != nil {
			e <- err
		} else {
			s <- size
		}
		close(s)
	}()

	size := 0
	select {
	case err := <-e:
		return err
	case str := <-s:
		str = strings.TrimSpace(str)
		parts := strings.SplitN(str, " ", 2)
		if len(parts) > 1 {
			return errors.Errorf("got error reply: %s", str)
		}
		size, err = strconv.Atoi(str)
		if err != nil {
			return err
		}
	case <-time.After(types.ReadTimeout):
		return errors.New("Read Timeout")
	}

	fmt.Printf("file size: %d\n", size)
	go func() {
		buf := make([]byte, size)
		remaining := size
		for {
			len, err := rw.Read(buf)
			if err != nil {
				e <- err
			}
			fmt.Printf("read %d bytes\n", len)
			if _, err := dst.Write(buf[:len]); err != nil {
				e <- err
			}
			remaining = remaining - len
			if remaining == 0 {
				break
			}
		}
		close(e)
	}()

	select {
	case err := <-e:
		return err
	case <-time.After(types.ReadTimeout):
		return errors.New("Read Timeout")
	}
	return nil
}

// PutRequest sends get request to remote peer
func (n *Node) PutRequest(ctx context.Context, pid peer.ID, content []byte, remoteDir string) error {
	if !path.IsAbs(remoteDir) {
		return errors.New("please use absolute path for remote path")
	}
	stream, err := n.host.NewStream(ctx, pid, types.PutURL)
	if err != nil {
		return err
	}
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	defer rw.Flush()

	if _, err := rw.WriteString(fmt.Sprintf("%d %s\n", len(content), remoteDir)); err != nil {
		return err
	}
	remaining := len(content)
	for {
		size, err := rw.Write(content)
		if err != nil {
			return err
		}
		fmt.Printf("Total legnth %d, write %d bytes\n", remaining, size)
		if size == remaining {
			return nil
		}
		remaining = remaining - size
		content = content[size:]
	}
	return nil
}
