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
	"github.com/stretchr/testify/assert"
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
	assert.NotEmpty(t, p1)
	assert.NotEmpty(t, p2)
}

// It creates a node, and then checks that the node is not nil.
func TestSetupSingleNode(t *testing.T) {
	p1, err1 := NewNode(NewNodeParams{
		Ctx: context.Background(),
	})
	if err1 != nil {
		t.Fatal(err1)
	}
	assert.NotEmpty(t, p1)
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
		Ctx:       context.Background(),
		Datastore: NewInMemoryDatastore(),
		Config:    cfg,
	}) // nil Datastore and Dht means we use the default
	if err1 != nil {
		t.Fatal(err1)
	}
	p2, err2 := NewNode(NewNodeParams{
		Ctx:       context.Background(),
		Datastore: NewInMemoryDatastore(),
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
	assert.NotEmpty(t, p1)
	assert.NotEmpty(t, p2)
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
	assert.Equal(t, "bafybeiawc5enlmxtwdbnts3mragh5eyhl3wn5qekvimw72igdj45lixbo4", node.Cid().String())
}

func TestAddPinFileAndGetItFromAnotherNode(t *testing.T) {
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
	p2, err1 := NewNode(NewNodeParams{
		Ctx:       context.Background(),
		Datastore: NewInMemoryDatastore(),
	})
	pinfo2 := peer.AddrInfo{
		ID:    p2.Host.ID(),
		Addrs: p2.Host.Addrs(),
	}
	if err1 != nil {
		t.Fatal(err1)
	}
	p1.BootstrapPeers([]peer.AddrInfo{pinfo2})
	p2.BootstrapPeers([]peer.AddrInfo{pinfo1})

	node, err := p1.AddPinFile(context.Background(), bytes.NewReader([]byte("letsrebuildtolearnnewthings!")), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("uploaded", node.Cid())
	assert.Equal(t, "bafybeiawc5enlmxtwdbnts3mragh5eyhl3wn5qekvimw72igdj45lixbo4", node.Cid().String())

	// now get the file from the other node
	node2, err := p2.GetFile(context.Background(), node.Cid())
	if err != nil {
		t.Fatal(err)
	}
	content2, err := io.ReadAll(node2)
	t.Log("retrieved node: ", string(content2))
	assert.Equal(t, "letsrebuildtolearnnewthings!", string(content2))
}

func TestGetFile(t *testing.T) {
	p1, err := NewNode(NewNodeParams{
		Ctx:       context.Background(),
		Datastore: NewInMemoryDatastore(),
	})
	node, err := p1.AddPinFile(context.Background(), bytes.NewReader([]byte("letsrebuildtolearnnewthings!")), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("uploaded", node.Cid())

	//cid, err := cid.Decode("bafybeiawc5enlmxtwdbnts3mragh5eyhl3wn5qekvimw72igdj45lixbo4")

	rsc, err := p1.GetFile(context.Background(), node.Cid())
	if err != nil {
		t.Fatal(err)
	}
	content2, err := io.ReadAll(rsc)
	t.Log("retrieved node: ", string(content2))
	assert.Equal(t, "letsrebuildtolearnnewthings!", string(content2))
}

func TestGetDirectory(t *testing.T) {
	p1, err := NewNode(NewNodeParams{
		Ctx:       context.Background(),
		Datastore: NewInMemoryDatastore(),
	})
	node, err := p1.AddPinDirectory(context.Background(), "./test/test_directory")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("uploaded", node.Cid())

	rsc, err := p1.GetDirectory(context.Background(), node)
	if err != nil {
		t.Fatal(err)
	}
	retrieveNode, err := rsc.GetNode()

	if err != nil {
		t.Fatal(err)
	}
	t.Log("retrieved root node", retrieveNode.Cid())
	assert.Equal(t, "bafybeihnhfwlfvq6eplc4i5cnj2of2whk6aab6kc4xeryr3ttfcaawjiyi", retrieveNode.Cid().String())
	assert.GreaterOrEqual(t, len(retrieveNode.Links()), 1)
}

func TestGetDirectoryWithCid(t *testing.T) {

	p1, err := NewNode(NewNodeParams{
		Ctx:       context.Background(),
		Datastore: NewInMemoryDatastore(),
	})
	p1.BootstrapPeers(DefaultBootstrapPeers())

	cid, err := cid.Decode("bafybeigvgzoolc3drupxhlevdp2ugqcrbcsqfmcek2zxiw5wctk3xjpjwy")
	rsc, err := p1.GetDirectoryWithCid(context.Background(), cid)
	if err != nil {
		t.Fatal(err)
	}
	retrieveNode, err := rsc.GetNode()

	if err != nil {
		t.Fatal(err)
	}
	t.Log("retrieved root node", retrieveNode.Cid())
	assert.Equal(t, "bafybeigvgzoolc3drupxhlevdp2ugqcrbcsqfmcek2zxiw5wctk3xjpjwy", retrieveNode.Cid().String())
	assert.GreaterOrEqual(t, len(retrieveNode.Links()), 1)
}

func TestAddPinDirectory(t *testing.T) {
	p1, err1 := NewNode(NewNodeParams{
		Ctx:       context.Background(),
		Datastore: NewInMemoryDatastore(),
	})
	if err1 != nil {
		t.Fatal(err1)
	}
	node, err := p1.AddPinDirectory(context.Background(), "./test/test_directory")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("uploaded root node", node.Cid())
	assert.NotEmpty(t, node)
	assert.Equal(t, "bafybeihnhfwlfvq6eplc4i5cnj2of2whk6aab6kc4xeryr3ttfcaawjiyi", node.Cid().String())
	rsc, err := p1.GetDirectory(context.Background(), node)
	if err != nil {
		t.Fatal(err)
	}
	retrieveNode, err := rsc.GetNode()

	if err != nil {
		t.Fatal(err)
	}
	t.Log("retrieved root node", retrieveNode.Cid())
	assert.Equal(t, "bafybeihnhfwlfvq6eplc4i5cnj2of2whk6aab6kc4xeryr3ttfcaawjiyi", retrieveNode.Cid().String())
	assert.GreaterOrEqual(t, len(retrieveNode.Links()), 1)
}

func TestAddPinDirectoryAndGetFromAnotherNode(t *testing.T) {
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
	p2, err1 := NewNode(NewNodeParams{
		Ctx:       context.Background(),
		Datastore: NewInMemoryDatastore(),
	})
	pinfo2 := peer.AddrInfo{
		ID:    p2.Host.ID(),
		Addrs: p2.Host.Addrs(),
	}
	if err1 != nil {
		t.Fatal(err1)
	}
	p1.BootstrapPeers([]peer.AddrInfo{pinfo2})
	p2.BootstrapPeers([]peer.AddrInfo{pinfo1})

	node, err := p1.AddPinDirectory(context.Background(), "./test/test_directory")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("uploaded root node", node.Cid())
	assert.NotEmpty(t, node)
	assert.Equal(t, "bafybeihnhfwlfvq6eplc4i5cnj2of2whk6aab6kc4xeryr3ttfcaawjiyi", node.Cid().String())
	rsc, err := p2.GetDirectory(context.Background(), node)
	if err != nil {
		t.Fatal(err)
	}
	retrieveNode, err := rsc.GetNode()

	if err != nil {
		t.Fatal(err)
	}
	t.Log("retrieved root node", retrieveNode.Cid())
	assert.Equal(t, "bafybeihnhfwlfvq6eplc4i5cnj2of2whk6aab6kc4xeryr3ttfcaawjiyi", retrieveNode.Cid().String())
	assert.GreaterOrEqual(t, len(retrieveNode.Links()), 1)

}

func TestOverrideDefaultConfig(t *testing.T) {
	params := NewNodeParams{
		Ctx:       context.Background(),
		Datastore: NewInMemoryDatastore(),
	}
	newConfig := &Config{
		Offline:           true,
		ReprovideInterval: 0,
		Libp2pKeyFile:     "mykey",
		ListenAddrs:       []string{"/ip4/127.0.0.1/tcp/0"},
		AnnounceAddrs:     nil,
		DatastoreDir: struct {
			Directory string
			Options   leveldb.Options
		}{},
		Blockstore:              "",
		NoBlockstoreCache:       false,
		NoAnnounceContent:       false,
		NoLimiter:               false,
		BitswapConfig:           BitswapConfig{},
		ConnectionManagerConfig: ConnectionManager{},
	}
	params.Config = params.ConfigurationBuilder(newConfig)

	p1, err1 := NewNode(params)
	if err1 != nil {
		t.Fatal(err1)
	}

	assert.Equal(t, p1.Config.Libp2pKeyFile, "mykey")
	assert.Equal(t, p1.Config.AnnounceAddrs, []string{"/ip4/0.0.0.0/tcp/0"})
	assert.Equal(t, p1.Config.ListenAddrs, []string{"/ip4/127.0.0.1/tcp/0"})
	assert.Equal(t, p1.Config.Offline, params.Config.Offline)
}
