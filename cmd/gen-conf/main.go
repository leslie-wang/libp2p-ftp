package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/leslie-wang/libp2p-ftp/types"

	crypto "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
)

func main() {
	//IPFS bootstrap nodes. Used to find other peers in the network.
	conf := types.Config{
		BootstrapNodes: []string{
			"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
			"/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
			"/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
			"/ip4/128.199.219.111/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
			"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		},
		RetryCount:    10,
		RetryInterval: time.Minute,
		HTTPListenPort: 8077,
	}
	// Set your own keypair
	priv, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	privBytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		log.Fatal(err)
	}
	conf.ServerPrivateKey = base64.StdEncoding.EncodeToString(privBytes)

	pubBytes, err := crypto.MarshalPublicKey(pub)
	if err != nil {
		log.Fatal(err)
	}
	conf.ServerPublicKey = base64.StdEncoding.EncodeToString(pubBytes)

	id, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		log.Fatal(err)
	}
	conf.ServerID = peer.IDB58Encode(id)

	f, err := os.Create("./conf.json")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	data, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.Write(data)
	if err != nil {
		log.Fatal(err)
	}

}
