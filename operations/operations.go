package operations

//"github.com/ipfs/ipfs-cluster/api/rest/client"
//ma "github.com/multiformats/go-multiaddr"

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	cruxIPFS "github.com/dedis/student_19_cruxIPFS"
	"github.com/ipfs/ipfs-cluster/api/rest/client"
	"go.dedis.ch/onet/v3/log"
)

var nodes = make(map[string]*Node)

// Read the given filename from the given node
func Read(node, filename string) map[string]time.Duration {
	if len(nodes) == 0 {
		nodes = LoadClusterInstances(cruxIPFS.SaveFile)
	}
	if n, ok := nodes[node]; ok {
		mutex := &sync.Mutex{}
		wg := sync.WaitGroup{}
		wg.Add(len(n.Clients))
		results := make(map[string]time.Duration)

		for i, c := range n.Clients {
			go func(c0 client.Client, m *sync.Mutex, i int) {
				t := readFile(c0, filename)
				m.Lock()
				results[n.Secrets[i]] = t
				m.Unlock()
				wg.Done()
			}(c, mutex, i)
		}
		wg.Wait()
		return results
	}
	panic(node + "do not exist")
}

// Write the given filename from the given node
func Write(node, filename string) (string, map[string]time.Duration) {
	if len(nodes) == 0 {
		nodes = LoadClusterInstances(cruxIPFS.SaveFile)
	}
	if n, ok := nodes[node]; ok {
		mutex := &sync.Mutex{}
		wg := sync.WaitGroup{}
		wg.Add(len(n.Clients))
		name := ""
		results := make(map[string]time.Duration)

		for i, c := range n.Clients {
			go func(c0 client.Client, m *sync.Mutex, i int) {
				n0, t := writeFile(c0, filepath.Join(fileFolder, filename))
				m.Lock()
				results[n.Secrets[i]] = t
				m.Unlock()
				if name == "" {
					name = n0
				}
				wg.Done()
			}(c, mutex, i)
		}
		wg.Wait()
		return name, results
	}
	panic(node + "do not exist")
}

func Write2(node, filename string) (string, map[string]time.Duration) {
	if len(nodes) == 0 {
		nodes = LoadClusterInstances(cruxIPFS.SaveFile)
	}
	if n, ok := nodes[node]; ok {
		mutex := &sync.Mutex{}
		wg := sync.WaitGroup{}
		wg.Add(len(n.Addrs))
		fmt.Println("len(n.Addrs)", len(n.Addrs))
		name := ""
		results := make(map[string]time.Duration)

		for i, host := range n.Addrs {
			func(h string, m *sync.Mutex, i int) {
				fmt.Println("Op with", h)
				n0, t := writeFile2(h, filepath.Join(fileFolder, filename))
				m.Lock()
				results[n.Secrets[i]] = t
				m.Unlock()
				if name == "" {
					name = n0
				}
				wg.Done()
			}(host, mutex, i)
		}
		fmt.Println("Waiting for hosts to finish")
		wg.Wait()
		return name, results
	}
	panic(node + "do not exist")

}

// NewFile write new file to disk
func NewFile(filename string) {
	os.Mkdir(fileFolder, defaultFileMode)
	str := strings.Repeat("abcd", 256)
	ioutil.WriteFile(filepath.Join(fileFolder, filename), []byte(str),
		defaultFileMode)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func printNodes() {
	str := ""
	for _, n := range nodes {
		str += n.Name + " : "
		for _, c := range n.Clients {
			str += fmt.Sprint(c, ", ")
		}
	}
	log.Lvl1(str)
}

// ListPeers of a client
func ListPeers(c client.Client) {
	ctx := context.Background()
	peers, err := c.Peers(ctx)
	checkErr(err)

	fmt.Printf("\nPeers in the Cluster:\n")
	for _, p := range peers {
		fmt.Printf("%s: %s\n", p.Peername, p.Addresses[0])
	}
}
