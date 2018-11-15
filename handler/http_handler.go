package handler

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/leslie-wang/libp2p-ftp/node"
	"github.com/leslie-wang/libp2p-ftp/types"
)

// HTTPHandler is the struct for handler request
type HTTPHandler struct {
	conf *types.Config
	node *node.Node
}

// NewHTTPHandler creates one handler
func NewHTTPHandler(c *types.Config) *HTTPHandler {
	return &HTTPHandler{conf: c}
}

// Close is to close handler and its corresponding host
func (h *HTTPHandler) Close() {
	h.node.Close()
}

// Serve starts node
func (h *HTTPHandler) Serve(ctx context.Context) error {
	if err := h.connect(ctx); err != nil {
		return err
	}

	go h.ping()

	http.HandleFunc(types.ListURL, h.list)
	http.HandleFunc(types.DeleteURL, h.delete)
	http.HandleFunc(types.GetURL, h.get)
	http.HandleFunc(types.PutURL, h.put)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", h.conf.HTTPListenPort), nil))

	select {}
}

func (h *HTTPHandler) connect(ctx context.Context) error {
	var err error
	h.node, err = node.StartNode(ctx, "", h.conf.BootstrapNodes)
	if err != nil {
		return err
	}

	return h.node.FindPeer(ctx, h.conf.ServerID)
}

func (h *HTTPHandler) ping() {
	failCount := 0
	for {
		time.Sleep(h.conf.RetryInterval)
		if failCount > 100 {
			// start reconnect
			if err := h.node.Close(); err != nil {
				fmt.Printf("node close got: %v\n", err)
				continue
			}
			if err := h.connect(context.Background()); err != nil {
				fmt.Printf("node connect got: %v\n", err)
				continue
			}
		}
		if err := h.node.PingRequest(context.Background()); err != nil {
			fmt.Printf("ping got: %v", err)
			failCount++
		} else {
			failCount = 0
		}
	}
}

func (h *HTTPHandler) list(w http.ResponseWriter, r *http.Request) {
	files, err := h.node.ListRequest(context.Background(), r.URL.Query().Get(types.QueryKeyDestination))
	if err != nil {
		writeError(w, err)
		return
	}

	w.Write([]byte(strings.Join(files, "\n")))
}

func (h *HTTPHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.node.DeleteRequest(context.Background(), r.URL.Query().Get(types.QueryKeyDestination)); err != nil {
		writeError(w, err)
		return
	}
}

func (h *HTTPHandler) get(w http.ResponseWriter, r *http.Request) {
	dst := r.URL.Query().Get(types.QueryKeyDestination)
	src := r.URL.Query().Get(types.QueryKeySource)
	filename := path.Base(dst)
	f, err := os.Create(path.Join(src, filename))
	if err != nil {
		writeError(w, err)
		return
	}
	defer f.Close()

	if err := h.node.GetRequest(context.Background(), dst, f); err != nil {
		writeError(w, err)
		return
	}
}

func (h *HTTPHandler) put(w http.ResponseWriter, r *http.Request) {
	dst := r.URL.Query().Get(types.QueryKeyDestination)
	src := r.URL.Query().Get(types.QueryKeySource)

	content, err := ioutil.ReadFile(src)
	if err != nil {
		writeError(w, err)
		return
	}

	if strings.HasSuffix(dst, "/") {
		dst = path.Join(dst, path.Base(src))
	}

	if err := h.node.PutRequest(context.Background(), content, dst); err != nil {
		writeError(w, err)
		return
	}
}

func writeError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
