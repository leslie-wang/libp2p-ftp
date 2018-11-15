package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/whyrusleeping/go-logging"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/leslie-wang/libp2p-ftp/handler"
	"github.com/leslie-wang/libp2p-ftp/types"

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
			Name:  "conf, c",
			Usage: "configure file name with whole path",
			Value: "/etc/libp2p-ftp/conf.json",
		},
		cli.IntFlag{
			Name:  "verbose",
			Usage: "log level: CRITICAL(0), ERROR(1), WARNING(2), NOTICE(3), INFO(4), DEBUG(5)",
			Value: 4,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:   "listen",
			Usage:  "listen as ftp server",
			Action: listen,
		},
		{
			Name:   "connect",
			Usage:  "connect to remote peer",
			Action: connect,
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

func loadConf(file string) (*types.Config, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	conf := &types.Config{}
	err = json.Unmarshal(content, conf)
	return conf, err
}

func listen(ctx *cli.Context) error {
	conf, err := loadConf(ctx.GlobalString("conf"))
	if err != nil {
		return err
	}

	h := handler.NewNodeHandler(conf)
	defer h.Close()

	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendLeveled := logging.AddModuleLevel(backend)
	backendLeveled.SetLevel(logging.Level(ctx.GlobalInt("verbose")), "")
	logging.SetBackend(backendLeveled)

	return h.Serve(context.Background())
}

func connect(ctx *cli.Context) error {
	conf, err := loadConf(ctx.GlobalString("conf"))
	if err != nil {
		return err
	}

	h := handler.NewHTTPHandler(conf)
	defer h.Close()

	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendLeveled := logging.AddModuleLevel(backend)
	backendLeveled.SetLevel(logging.Level(ctx.GlobalInt("verbose")), "")
	logging.SetBackend(backendLeveled)

	return h.Serve(context.Background())
}

func list(cctx *cli.Context) error {
	if len(cctx.Args()) < 1 {
		return errors.New("Invalid number of arguments")
	}
	conf, err := loadConf(cctx.GlobalString("conf"))
	if err != nil {
		return err
	}
	resp, err := httpRequest(fmt.Sprintf("http://localhost:%d%s?%s=%s", conf.HTTPListenPort, types.ListURL, types.QueryKeyDestination, cctx.Args()[0]))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}

func put(cctx *cli.Context) error {
	if len(cctx.Args()) != 2 {
		return errors.New("Invalid number of arguments")
	}

	if !path.IsAbs(cctx.Args()[0]) {
		return errors.New("please use absolute destination path\n")
	}
	if !path.IsAbs(cctx.Args()[1]) {
		return errors.New("please use absolute source path\n")
	}

	conf, err := loadConf(cctx.GlobalString("conf"))
	if err != nil {
		return err
	}
	_, err = httpRequest(fmt.Sprintf("http://localhost:%d%s?%s=%s&%s=%s", conf.HTTPListenPort, types.PutURL,
		types.QueryKeyDestination, cctx.Args()[1], types.QueryKeySource, cctx.Args()[0]))
	return err
}

func get(cctx *cli.Context) error {
	if len(cctx.Args()) != 2 {
		return errors.New("Invalid number of arguments")
	}

	if !path.IsAbs(cctx.Args()[0]) {
		return errors.New("please use absolute destination path\n")
	}
	if !path.IsAbs(cctx.Args()[1]) {
		return errors.New("please use absolute source path\n")
	}

	conf, err := loadConf(cctx.GlobalString("conf"))
	if err != nil {
		return err
	}
	_, err = httpRequest(fmt.Sprintf("http://localhost:%d%s?%s=%s&%s=%s", conf.HTTPListenPort, types.GetURL,
		types.QueryKeyDestination, cctx.Args()[0], types.QueryKeySource, cctx.Args()[1]))
	return err
}

func delete(cctx *cli.Context) error {
	if len(cctx.Args()) != 1 {
		return errors.New("Invalid number of arguments")
	}
	conf, err := loadConf(cctx.GlobalString("conf"))
	if err != nil {
		return err
	}
	_, err = httpRequest(fmt.Sprintf("http://localhost:%d%s?%s=%s", conf.HTTPListenPort, types.DeleteURL, types.QueryKeyDestination, cctx.Args()[0]))
	return err
}

func httpRequest(url string) (*http.Response, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("Non 200 reply: %s", string(data))
	}
	return resp, nil
}
