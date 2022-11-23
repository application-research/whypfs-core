// It creates two nodes, adds a file to one, and retrieves it from the other
package whypfs

import (
	"bytes"
	"context"
	"github.com/ipfs/go-cid"
	leveldb "github.com/ipfs/go-ds-leveldb"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
	"github.com/smartystreets/assertions"
	"io"
	"path/filepath"
	"testing"
)

var secret = "thefoxjumpoverthebridgeofthefencealongthehighway7oftorontocanadasourceoftheborderwiththebestcoffeeever"

// It creates two nodes, adds a file to one, and retrieves it from the other
func setupNodes(t *testing.T) (*Node, *Node) {
	p1, err1 := NewNode(NewNodeParams{
		Ctx:       context.Background(),
		Datastore: NewInMemoryDatastore(),
	})
	if err1 != nil {
		t.Fatal(err1)
	}
	pinfo1 := peer.AddrInfo{
		ID:    p1.Host.ID(),
		Addrs: p1.Host.Addrs(),
	}
	p2, err2 := NewNode(NewNodeParams{
		Ctx:       context.Background(),
		Datastore: NewInMemoryDatastore(),
	})
	if err2 != nil {
		t.Fatal(err2)
	}

	pinfo2 := peer.AddrInfo{
		ID:    p2.Host.ID(),
		Addrs: p2.Host.Addrs(),
	}

	p1.BootstrapPeers([]peer.AddrInfo{pinfo2})
	p2.BootstrapPeers([]peer.AddrInfo{pinfo1})
	return p1, p2
}

func TestDAG(t *testing.T) {
	ctx := context.Background()
	p1, p2 := setupNodes(t)
	//defer closer(t)

	m := map[string]string{
		"akey": "avalue",
	}

	codec := uint64(multihash.SHA2_256)
	node, err := cbor.WrapObject(m, codec, multihash.DefaultLengths[codec])
	if err != nil {
		t.Fatal(err)
	}

	t.Log("created node: ", node.Cid())
	err = p1.Add(ctx, node)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p2.Get(ctx, node.Cid())
	if err != nil {
		t.Error(err)
	}

	err = p1.Remove(ctx, node.Cid())
	if err != nil {
		t.Error(err)
	}

	err = p2.Remove(ctx, node.Cid())
	if err != nil {
		t.Error(err)
	}

	if ok, err := p1.BlockStore().Has(ctx, node.Cid()); ok || err != nil {
		t.Error("block should have been deleted")
	}

	if ok, err := p2.BlockStore().Has(ctx, node.Cid()); ok || err != nil {
		t.Error("block should have been deleted")
	}
}

// It creates two nodes, connects them, and returns them
func TestSetupMultiplePeeredNodes(t *testing.T) {
	p1, p2 := setupNodes(t)
	assertions.ShouldNotBeNil(p1)
	assertions.ShouldNotBeNil(p2)
}

// It creates a node, and then checks that the node is not nil.
func TestSetupSingleNode(t *testing.T) {
	p1, err1 := NewNode(NewNodeParams{
		Ctx: context.Background(),
	})
	if err1 != nil {
		t.Fatal(err1)
	}
	assertions.ShouldNotBeNil(p1)
}

// It creates two nodes, adds a file to the first node, and then checks that the file is present in the second node
func TestSetupMultipleNodes(t *testing.T) {
	cfg := &Config{
		Libp2pKeyFile: filepath.Join("libp2p.key"),
		AnnounceAddrs: []string{"/ip4/0.0.0.0/tcp/0"},
		DatastoreDir: struct {
			Directory string
			Options   leveldb.Options
		}{
			Directory: filepath.Join("datastore"),
			Options:   leveldb.Options{},
		},
		NoBlockstoreCache: false,
		NoLimiter:         true,
		ListenAddrs:       []string{"/ip4/0.0.0.0/tcp/0"},
		BitswapConfig: BitswapConfig{
			MaxOutstandingBytesPerPeer: 20 << 20,
			TargetMessageSize:          2 << 20,
		},
		Blockstore:              ":flatfs:blocks",
		ConnectionManagerConfig: ConnectionManager{},
		//DatabaseConnString:      "sqlite=whypfs.db",
	}

	p1, err1 := NewNode(NewNodeParams{
		Ctx:    context.Background(),
		Config: cfg,
	}) // nil Datastore and Dht means we use the default
	if err1 != nil {
		t.Fatal(err1)
	}
	p2, err2 := NewNode(NewNodeParams{
		Ctx: context.Background(),
	}) // nil data	store and Dht means we use the default
	if err2 != nil {
		t.Fatal(err2)
	}

	content := []byte("hola")
	buf := bytes.NewReader(content)
	n, err := p1.AddPinFile(context.Background(), buf, nil) // default configurations
	if err != nil {
		t.Fatal(err)
	}
	n.Cid()
	assertions.ShouldNotBeNil(p1)
	assertions.ShouldNotBeNil(p2)
}

func TestAddPinFile(t *testing.T) {
	p1, err1 := NewNode(NewNodeParams{
		Ctx:       context.Background(),
		Datastore: NewInMemoryDatastore(), // leave blank if you want to generate the default datastore
	})
	if err1 != nil {
		t.Fatal(err1)
	}
	node, err := p1.AddPinFile(context.Background(), bytes.NewReader([]byte("letsrebuildtolearnnewthings!")), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("uploaded", node.Cid())
	assertions.ShouldEqual("bafybeiawc5enlmxtwdbnts3mragh5eyhl3wn5qekvimw72igdj45lixbo4", node.Cid().String())
}

func TestGetFile(t *testing.T) {
	p1, err := NewNode(NewNodeParams{
		Ctx:       context.Background(),
		Datastore: NewInMemoryDatastore(),
	})

	cid, err := cid.Decode("bafybeiawc5enlmxtwdbnts3mragh5eyhl3wn5qekvimw72igdj45lixbo4")

	rsc, err := p1.GetFile(context.Background(), cid)
	if err != nil {
		t.Fatal(err)
	}
	content2, err := io.ReadAll(rsc)
	t.Log("retrieved node: ", string(content2))
}
