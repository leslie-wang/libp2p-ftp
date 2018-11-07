package main

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p-peer"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/leslie-wang/p2pftp/node"
	"github.com/pkg/errors"

	"github.com/urfave/cli"
)

// IPFS bootstrap nodes. Used to find other peers in the network.
var bootstrapPeers = []string{
	"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	"/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
	"/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
	"/ip4/128.199.219.111/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
	"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
}

func main() {
	app := cli.NewApp()

	app.Usage = "simple p2pftp application"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "rendezvous, r",
			Usage: "Unique string to identify group of nodes. Share this with your friends to let them connect with you",
			Value: "p2p_ftp",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:   "listen",
			Usage:  "listen as ftp server",
			Action: listen,
		},
		{
			Name:      "list",
			ArgsUsage: "[dir name]",
			Usage:     "list files under given directory",
			Action:    list,
		},
		{
			Name:      "put",
			ArgsUsage: "[local filename] [remote dir]",
			Usage:     "put file name to remote directory",
			Action:    put,
		},
		{
			Name:      "get",
			ArgsUsage: "[remote filename] [local dir]",
			Usage:     "get remote file",
			Action:    get,
		},
		{
			Name:      "delete",
			ArgsUsage: "[filename]",
			Usage:     "delete remote file",
			Action:    delete,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func listen(ctx *cli.Context) error {
	_, err := node.StartNode(context.Background(), ctx.GlobalString("rendezvous"), bootstrapPeers, true)
	if err != nil {
		return err
	}

	select {}
}

func list(cctx *cli.Context) error {
	if len(cctx.Args()) < 1 {
		return errors.New("Invalid number of arguments")
	}
	ctx := context.Background()
	n, err := node.StartNode(ctx, cctx.GlobalString("rendezvous"), bootstrapPeers, false)
	if err != nil {
		return err
	}

	return request(ctx, n, func(ctx context.Context, id peer.ID) error {
		list, err := n.ListRequest(ctx, id, cctx.Args()[0])
		if err != nil {
			return err
		}
		for _, f := range list {
			fmt.Println(f)
		}
		return nil
	})
}

func put(cctx *cli.Context) error {
	if len(cctx.Args()) != 2 {
		return errors.New("Invalid number of arguments")
	}

	content, err := ioutil.ReadFile(cctx.Args()[0])
	if err != nil {
		return err
	}

	remoteFile := cctx.Args()[1]
	if strings.HasSuffix(remoteFile, "/") {
		remoteFile = path.Join(remoteFile, path.Base(cctx.Args()[0]))
	}


	ctx := context.Background()
	n, err := node.StartNode(ctx, cctx.GlobalString("rendezvous"), bootstrapPeers,false)
	if err != nil {
		return err
	}

	return request(ctx, n, func(ctx context.Context, id peer.ID) error {
		return n.PutRequest(ctx, id, content, remoteFile)
	})
}

func get(cctx *cli.Context) error {
	if len(cctx.Args()) != 2 {
		return errors.New("Invalid number of arguments")
	}

	var f *os.File
	filename := path.Base(cctx.Args()[0])
	if path.IsAbs(cctx.Args()[1]) {
		var err error
		f, err = os.Create(path.Join(cctx.Args()[1], filename))
		if err != nil {
			return err
		}
	} else {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		localDir := path.Join(wd, cctx.Args()[1])
		if err := os.MkdirAll(localDir, 0700); err != nil {
			return err
		}
		f, err = os.Create(path.Join(localDir, filename))
		if err != nil {
			return err
		}
	}
	defer f.Close()

	ctx := context.Background()
	n, err := node.StartNode(ctx, cctx.GlobalString("rendezvous"), bootstrapPeers,false)
	if err != nil {
		return err
	}

	return request(ctx, n, func(ctx context.Context, id peer.ID) error {
		return n.GetRequest(ctx, id, cctx.Args()[0], f)
	})
}

func delete(cctx *cli.Context) error {
	if len(cctx.Args()) < 1 {
		return errors.New("Invalid number of arguments")
	}
	ctx := context.Background()
	n, err := node.StartNode(ctx, cctx.GlobalString("rendezvous"), bootstrapPeers,false)
	if err != nil {
		return err
	}

	return request(ctx, n, func(ctx context.Context, id peer.ID) error {
		return n.DeleteRequest(ctx, id, cctx.Args()[0])
	})
}

func request(ctx context.Context, n *node.Node, request func (ctx context.Context, id peer.ID) error) error {
	for i := 0; i < 10; i++ {
		peers, err := n.DiscoverPeers(ctx)
		if err != nil {
			fmt.Println(err)
			continue
		}
		for _, p := range peers {
			if p.ID == n.ID() || len(p.Addrs) == 0 {
				// No sense connecting to ourselves or if addrs are not available
				continue
			}

			if err := request(ctx, p.ID); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("request success")
				return nil
			}
		}

		fmt.Println("Unable read, sleep 1 minute and try again")
		time.Sleep(time.Minute)
	}
	return nil
}