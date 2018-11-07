package handler

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/leslie-wang/p2pftp/types"

	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
)

// Handler is the struct for handler request
type Handler struct {
	host host.Host
}

// NewHandler creates one handler
func NewHandler(h host.Host) *Handler {
	return &Handler{host: h}
}

// Close is to close handler and its corresponding host
func (h *Handler) Close() {
}

// MkRoutes creates route handler
func (h *Handler) MkRoutes() {
	h.host.SetStreamHandler(types.ListURL, list)
	h.host.SetStreamHandler(types.DeleteURL, delete)
	h.host.SetStreamHandler(types.GetURL, get)
	h.host.SetStreamHandler(types.PutURL, put)
}

func list(stream inet.Stream) {
	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	defer rw.Flush()

	dir, err := rw.ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Printf("list request: %s", dir)

	if !path.IsAbs(dir) {
		if _, err := rw.WriteString("please use absolute path\n"); err != nil {
			fmt.Println(err)
		}
	}

	files, err := ioutil.ReadDir(strings.TrimSpace(dir))
	if err != nil {
		if _, err := rw.WriteString(fmt.Sprintf("%s\n", err.Error())); err != nil {
			fmt.Println(err)
		}
	} else {
		for _, file := range files {
			if _, err := rw.WriteString(fmt.Sprintf("%s\n", file.Name())); err != nil {
				fmt.Println(err)
			}
		}
	}

	if _, err := rw.WriteString("\n"); err != nil {
		fmt.Println(err)
	}
}

func delete(stream inet.Stream) {
	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	defer rw.Flush()

	dir, err := rw.ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Printf("delete request: %s", dir)
	if !path.IsAbs(dir) {
		if _, err := rw.WriteString("please use absolute path\n"); err != nil {
			fmt.Println(err)
		}
	}

	if err := os.RemoveAll(strings.TrimSpace(dir)); err != nil {
		if _, err := rw.WriteString(fmt.Sprintf("%s\n\n", err.Error())); err != nil {
			fmt.Println(err)
		}
		return
	}
	if _, err := rw.WriteString("\n\n"); err != nil {
		fmt.Println(err)
	}
}

func get(stream inet.Stream) {
	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	defer rw.Flush()

	file, err := rw.ReadString('\n')
	if err != nil {
		fmt.Println(err)
		if _, err := rw.WriteString(fmt.Sprintf("-1 %s", err.Error())); err != nil {
			fmt.Println(err)
		}
		return
	}
	log.Printf("get request: %s", file)
	file = strings.TrimSpace(file)
	if !path.IsAbs(file) {
		if _, err := rw.WriteString("-1 please use absolute path"); err != nil {
			fmt.Println(err)
		}
	}

	info, err := os.Stat(file)
	if err != nil {
		fmt.Println(err)
		if _, err := rw.WriteString(fmt.Sprintf("-1 %s", err.Error())); err != nil {
			fmt.Println(err)
		}
		return
	}
	if !info.Mode().IsRegular() {
		if _, err := rw.WriteString("-1 remote path is not regular file"); err != nil {
			fmt.Println(err)
		}
		return
	}

	if _, err := rw.WriteString(fmt.Sprintf("%d\n", info.Size())); err != nil {
		fmt.Println(err)
		return
	}

	content, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err)
		return
	}

	remaining := len(content)
	for {
		size, err := rw.Write(content)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Total legnth %d, write %d bytes\n", remaining, size)
		if size == remaining {
			return
		}
		remaining = remaining - size
		content = content[size:]
	}
}

func put(stream inet.Stream) {
	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	defer rw.Flush()

	line, err := rw.ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return
	}
	line = strings.TrimSpace(line)
	log.Printf("get request: %s", line)
	parts := strings.SplitN(line, " ", 2)

	size, err := strconv.Atoi(parts[0])
	if err != nil {
		fmt.Println(err)
		return
	}

	if err := os.MkdirAll(path.Dir(parts[1]), 0700); err != nil {
		fmt.Println(err)
		return
	}

	f, err := os.Create(parts[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	e := make(chan error)

	go func() {
		buf := make([]byte, size)
		remaining := size
		for {
			len, err := rw.Read(buf)
			if err != nil {
				e <- err
			}
			if _, err := f.Write(buf[:len]); err != nil {
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
		if err != nil {
			fmt.Println(err)
		}
		return
	case <-time.After(types.ReadTimeout):
		fmt.Println("Read Timeout")
	}
}
