package whypfs

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	ulimit "github.com/filecoin-project/go-ulimit"
	"github.com/ipfs/go-bitswap"
	bsnet "github.com/ipfs/go-bitswap/network"
	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-cidutil"
	"github.com/ipfs/go-datastore"
	flatfs "github.com/ipfs/go-ds-flatfs"
	levelds "github.com/ipfs/go-ds-leveldb"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	chunker "github.com/ipfs/go-ipfs-chunker"
	exchange "github.com/ipfs/go-ipfs-exchange-interface"
	provider "github.com/ipfs/go-ipfs-provider"
	"github.com/ipfs/go-ipfs-provider/queue"
	"github.com/ipfs/go-ipfs-provider/simple"
	cbor "github.com/ipfs/go-ipld-cbor"
	ipld "github.com/ipfs/go-ipld-format"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipfs/go-merkledag"
	metri "github.com/ipfs/go-metrics-interface"
	"github.com/ipfs/go-unixfs/importer/balanced"
	"github.com/ipfs/go-unixfs/importer/helpers"
	"github.com/ipfs/go-unixfs/importer/trickle"
	ufsio "github.com/ipfs/go-unixfs/io"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/fullrt"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	metrics "github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
	bsm "github.com/whyrusleeping/go-bs-measure"
	"golang.org/x/xerrors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var logger = logging.Logger("whypfs-core")

var (
	defaultReprovideInterval = 8 * time.Hour
)
var BootstrapPeers []peer.AddrInfo

func init() {
	ipld.Register(cid.DagProtobuf, merkledag.DecodeProtobufBlock)
	ipld.Register(cid.Raw, merkledag.DecodeRawBlock)
	ipld.Register(cid.DagCBOR, cbor.DecodeBlock)
}

type NewNodeParams struct {
	// Context is the context to use for the node.
	Ctx        context.Context
	Repo       string
	Datastore  datastore.Batching
	Blockstore blockstore.Blockstore
	Dht        *dht.IpfsDHT
	Config     *Config
}

type Node struct {
	//	node context
	Ctx context.Context

	//	Node configuration
	Config *Config

	// hosts
	Host       host.Host
	Dht        *dht.IpfsDHT
	StorageDir string

	// dag service
	ipld.DAGService
	Blockstore   blockstore.Blockstore
	Blockservice blockservice.BlockService
	Datastore    datastore.Batching
	Reprovider   provider.System
	Exchange     exchange.Interface
	Bitswap      *bitswap.Bitswap
	FullRt       *fullrt.FullRT
}

type Config struct {
	// The DAGService will not announce or retrieve blocks from the network
	Offline bool
	// ReprovideInterval sets how often to reprovide records to the DHT
	ReprovideInterval time.Duration
	Libp2pKeyFile     string
	ListenAddrs       []string
	AnnounceAddrs     []string
	DatastoreDir      struct {
		Directory string
		Options   levelds.Options
	}
	Blockstore              string
	NoBlockstoreCache       bool
	NoAnnounceContent       bool
	NoLimiter               bool
	BitswapConfig           BitswapConfig
	ConnectionManagerConfig ConnectionManager
}

type ConnectionManager struct {
	HighWater int
	LowWater  int
}

type BitswapConfig struct {
	MaxOutstandingBytesPerPeer int64
	TargetMessageSize          int
}

func (cfg *Config) setDefaults() {

	// optimal settings
	cfg.Offline = false
	cfg.ReprovideInterval = defaultReprovideInterval
	cfg.NoBlockstoreCache = false
	cfg.NoAnnounceContent = false
	cfg.NoLimiter = false
	cfg.BitswapConfig.MaxOutstandingBytesPerPeer = 1 << 20
	cfg.BitswapConfig.TargetMessageSize = 1 << 20
	cfg.ConnectionManagerConfig.HighWater = 1000
	cfg.ConnectionManagerConfig.LowWater = 900
	cfg.DatastoreDir.Directory = "datastore"
	cfg.DatastoreDir.Options = levelds.Options{}
	cfg.Blockstore = ":flatfs:.whypfs/blocks"
	cfg.Libp2pKeyFile = filepath.Join("libp2p.key")
	cfg.ListenAddrs = []string{"/ip4/0.0.0.0/tcp/0"}
	cfg.AnnounceAddrs = []string{"/ip4/0.0.0.0/tcp/0"}
}

//	NewNode creates a new WhyPFS node with the given configuration.
func NewNode(
	nodeParams NewNodeParams) (*Node, error) {
	var err error
	if nodeParams.Config == nil {
		nodeParams.Config = &Config{}
		nodeParams.Config.setDefaults()
	}

	if nodeParams.Repo == "" {
		nodeParams.Repo = ".whypfs"
	}

	ch, nlim, err := ulimit.ManageFdLimit(50000)
	if err != nil {
		return nil, err
	}

	if ch {
		logger.Infof("changed file descriptor limit to %d", nlim)
	}

	if err = ensureRepoExists(nodeParams.Repo); err != nil {
		return nil, err
	}

	nodeParams.Config.Blockstore = ":flatfs:" + filepath.Join(nodeParams.Repo, "blocks")

	// create the node
	node := &Node{}
	node.Config = nodeParams.Config
	node.Ctx = nodeParams.Ctx
	node.Datastore = nodeParams.Datastore
	node.Dht = nodeParams.Dht

	// set up defaults here.
	node.setupPeer() // peer Host

	// if they don't have a datastore, let's set it up for them.
	err = node.setupDatastore()
	if err != nil {
		return nil, err
	}

	err = node.setupBlockstore() // Blockstore and service
	if err != nil {
		return nil, err
	}
	err = node.setupBlockservice() // block service
	if err != nil {
		return nil, err
	}
	err = node.setupDAGService() // DAG service
	if err != nil {
		node.Blockservice.Close()
		return nil, err
	}

	err = node.setupReprovider() // Reprovider
	if err != nil {
		node.Blockservice.Close()
		return nil, err
	}

	//	return the node
	go node.deferClose()
	return node, nil
}
func ensureRepoExists(dir string) error {
	st, err := os.Stat(dir)
	if err == nil {
		if st.IsDir() {
			return nil
		}
		return fmt.Errorf("repo dir was not a directory")
	}

	if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return nil
}

//	Helper function to bootstrap peers
// Bootstrapping the node to the network.
func (p *Node) BootstrapPeers(peers []peer.AddrInfo) {
	connected := make(chan struct{})
	var wg sync.WaitGroup
	for _, pinfo := range peers {
		wg.Add(1)
		go func(pinfo peer.AddrInfo) {
			defer wg.Done()
			err := p.Host.Connect(p.Ctx, pinfo)
			if err != nil {
				logger.Warn(err)
				return
			}
			logger.Info("Connected to", pinfo.ID)
			connected <- struct{}{}
		}(pinfo)
	}

	go func() {
		wg.Wait()
		close(connected)
	}()

	i := 0
	for range connected {
		i++
	}
	if nPeers := len(peers); i < nPeers/2 {
		logger.Warnf("only connected to %d bootstrap peers out of %d", i, nPeers)
	}

	err := p.Dht.Bootstrap(p.Ctx)
	if err != nil {
		logger.Error(err)
		return
	}
}

// Creating a new function called Session that takes in a context and returns a NodeGetter.
func (p *Node) Session(ctx context.Context) ipld.NodeGetter {
	ng := merkledag.NewSession(ctx, p.DAGService)
	if ng == p.DAGService {
		logger.Warn("DAGService does not support sessions")
	}
	return ng
}

// AddParams contains all of the configurable parameters needed to specify the
// importing process of a file.
type AddParams struct {
	Layout    string
	Chunker   string
	RawLeaves bool
	Hidden    bool
	Shard     bool
	NoCopy    bool
	HashFun   string
}

// BlockStore offers access to the Blockstore underlying the Peer's DAGService.
func (p *Node) BlockStore() blockstore.Blockstore {
	return p.Blockstore
}

// HasBlock returns whether a given block is available locally. It is
// a shorthand for .Blockstore().Has().
func (p *Node) HasBlock(ctx context.Context, c cid.Cid) (bool, error) {
	return p.BlockStore().Has(ctx, c)
}

// Setting up the node.
func (p *Node) setupPeer() error {

	//	 libp2p peer key
	var cryptoPrivateKey crypto.PrivKey
	data, err := ioutil.ReadFile(p.Config.Libp2pKeyFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		k, _, err := crypto.GenerateEd25519Key(crand.Reader)
		if err != nil {
			return err
		}

		data, err := crypto.MarshalPrivateKey(k)
		if err != nil {
			return err
		}

		if err := ioutil.WriteFile(p.Config.Libp2pKeyFile, data, 0600); err != nil {
			return err
		}
		cryptoPrivateKey = k
	} else {
		cryptoPrivateKey, err = crypto.UnmarshalPrivateKey(data)

		if err != nil {
			return err
		}
	}

	var rcm network.ResourceManager
	if p.Config.NoLimiter || true {
		rcm, err = network.NullResourceManager, nil
		logger.Warnf("starting node with no resource limits")
	}

	if err != nil {
		return err
	}

	bwc := metrics.NewBandwidthCounter()
	cmgr, err := connmgr.NewConnManager(p.Config.ConnectionManagerConfig.LowWater, p.Config.ConnectionManagerConfig.HighWater)
	if err != nil {
		return err
	}
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(p.Config.ListenAddrs...),
		libp2p.NATPortMap(),
		libp2p.ConnectionManager(cmgr),
		libp2p.Identity(cryptoPrivateKey),
		libp2p.BandwidthReporter(bwc),
		libp2p.DefaultTransports,
		libp2p.ResourceManager(rcm),
	}

	if len(p.Config.AnnounceAddrs) > 0 {
		var addrs []multiaddr.Multiaddr
		for _, anna := range p.Config.AnnounceAddrs {
			a, err := multiaddr.NewMultiaddr(anna)
			if err != nil {
				return fmt.Errorf("failed to parse announce addr: %w", err)
			}
			addrs = append(addrs, a)
		}
		opts = append(opts, libp2p.AddrsFactory(func([]multiaddr.Multiaddr) []multiaddr.Multiaddr {
			return addrs
		}))
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return err
	}
	p.Host = h
	return nil
}

// Setting up the datastore, blockstore, blockservice, and reprovider.
func (p *Node) setupDatastore() error {

	//	if it's nil, then let's set it up for them.
	if p.Datastore == nil {
		ds, err := levelds.NewDatastore(p.Config.DatastoreDir.Directory, &p.Config.DatastoreDir.Options)
		if err != nil {
			return err
		}
		p.Datastore = ds // data store.

		dhtopts := fullrt.DHTOption(
			dht.Datastore(p.Datastore),
			dht.BootstrapPeers(DefaultBootstrapPeers()...),
			dht.BucketSize(20),
		)

		frt, err := fullrt.NewFullRT(p.Host, dht.DefaultPrefix, dhtopts)
		if err != nil {
			return xerrors.Errorf("constructing fullrt: %w", err)
		}
		p.FullRt = frt // full routing table

	}

	//	no ipfs Dht, let's set it up for them.
	if p.Dht == nil {
		ipfsdht, err := dht.New(p.Ctx, p.Host, dht.Datastore(p.Datastore))
		if err != nil {
			return xerrors.Errorf("constructing Dht: %w", err)
		}
		p.Dht = ipfsdht // ipfs Dht
	}

	return nil
}

// Setting up the DAG service.
func (p *Node) setupDAGService() error {
	p.DAGService = merkledag.NewDAGService(p.Blockservice)
	return nil
}

// Setting up the blockstore.
func (p *Node) setupBlockstore() error {
	mbs, storedir, err := loadBlockstore(p.Config.Blockstore, p.Config.NoBlockstoreCache)
	if err != nil {
		return err
	}

	p.StorageDir = storedir
	p.Blockstore = mbs
	return nil
}

// Setting up the Bitswap network and the Bitswap service.
func (p *Node) setupBlockservice() error {

	//	Bitswap network
	bswapnet := bsnet.NewFromIpfsHost(p.Host, p.Dht)

	peerwork := p.Config.BitswapConfig.MaxOutstandingBytesPerPeer
	if peerwork == 0 {
		peerwork = 5 << 20
	}

	bsopts := []bitswap.Option{
		bitswap.EngineBlockstoreWorkerCount(600),
		bitswap.TaskWorkerCount(600),
		bitswap.MaxOutstandingBytesPerPeer(int(peerwork)),
	}

	if tms := p.Config.BitswapConfig.TargetMessageSize; tms != 0 {
		bsopts = append(bsopts, bitswap.WithTargetMessageSize(tms))
	}

	//	Bitswap
	bswap := bitswap.New(p.Ctx, bswapnet, p.Blockstore, bsopts...)
	p.Blockservice = blockservice.New(p.Blockstore, bswap)
	p.Bitswap = bswap

	return nil
}

// Setting up the Reprovider.
func (p *Node) setupReprovider() error {
	if p.Config.Offline || p.Config.ReprovideInterval < 0 {
		p.Reprovider = provider.NewOfflineProvider()
		return nil
	}

	queue, err := queue.NewQueue(p.Ctx, "provq", p.Datastore)
	if err != nil {
		return err
	}

	prov := simple.NewProvider(
		p.Ctx,
		queue,
		p.Dht,
	)

	reprov := simple.NewReprovider(
		p.Ctx,
		p.Config.ReprovideInterval,
		p.Dht,
		simple.NewBlockstoreProvider(p.Blockstore),
	)

	p.Reprovider = provider.NewSystem(prov, reprov)
	p.Reprovider.Run()
	return nil
}

type deleteManyWrap struct {
	blockstore.Blockstore
}

func (dmw *deleteManyWrap) DeleteMany(ctx context.Context, cids []cid.Cid) error {
	for _, c := range cids {
		if err := dmw.Blockstore.DeleteBlock(ctx, c); err != nil {
			return err
		}
	}

	return nil
}

// It takes a blockstore configuration string, and returns a blockstore.Blockstore, a string representing the directory of
// the blockstore, and an error
func loadBlockstore(bscfg string, nocache bool) (blockstore.Blockstore, string, error) {
	bstore, dir, err := constructBlockstore(bscfg)
	if err != nil {
		return nil, "", err
	}

	ctx := metri.CtxScope(context.TODO(), "estuary.bstore")

	bstore = bsm.New("estuary.blks.base", bstore)

	if !nocache {
		cbstore, err := blockstore.CachedBlockstore(ctx, bstore, blockstore.CacheOpts{
			//HasBloomFilterSize:   512 << 20,
			//HasBloomFilterHashes: 7,
			HasARCCacheSize: 8 << 20,
		})
		if err != nil {
			return nil, "", err
		}
		bstore = &deleteManyWrap{cbstore}
	}

	mbs := bsm.New("estuary.repo", bstore)

	var blkst blockstore.Blockstore = mbs

	return blkst, dir, nil
}

type DeleteManyBlockstore interface {
	blockstore.Blockstore
	DeleteMany(context.Context, []cid.Cid) error
	// It parses the blockstore configuration string, and then it creates a blockstore based on the configuration
}

func constructBlockstore(bscfg string) (DeleteManyBlockstore, string, error) {

	spec, params, path, err := parseBsCfg(bscfg)
	if err != nil {
		return nil, "", err
	}

	switch spec {
	case "flatfs":
		sfs := "/repo/flatfs/shard/v1/next-to-last/3"
		if len(params) > 0 {
			parts := strings.Split(params[0], "=")
			switch parts[0] {
			case "type":
				switch parts[1] {
				case "estuary":
					// default
					sfs = "/repo/flatfs/shard/v1/next-to-last/3"
				case "go-ipfs":
					sfs = "/repo/flatfs/shard/v1/next-to-last/2"
				default:
					return nil, "", fmt.Errorf("unrecognized flatfs repo type in params: %s", parts[1])
				}
			}
		}
		sf, err := flatfs.ParseShardFunc(sfs)
		if err != nil {
			return nil, "", err
		}

		ds, err := flatfs.CreateOrOpen(path, sf, false)
		if err != nil {
			return nil, "", err
		}

		return &deleteManyWrap{blockstore.NewBlockstoreNoPrefix(ds)}, path, nil

	default:
		return nil, "", fmt.Errorf("unrecognized Blockstore spec: %q", spec)
	}
}

func parseBsCfg(bscfg string) (string, []string, string, error) {
	if bscfg[0] != ':' {
		return "", nil, "", fmt.Errorf("Config must start with colon")
	}

	var inParen bool
	var parenStart int
	var parenEnd int
	var end int
	for i := 1; i < len(bscfg); i++ {
		if inParen {
			if bscfg[i] == ')' {
				inParen = false
				parenEnd = i
			}
			continue
		}

		if bscfg[i] == '(' {
			inParen = true
			parenStart = i
		}

		if bscfg[i] == ':' {
			end = i
			break
		}
	}

	if parenStart == 0 {
		return bscfg[1:end], nil, bscfg[end+1:], nil
	}

	t := bscfg[1:parenStart]
	params := strings.Split(bscfg[parenStart+1:parenEnd], ",")

	return t, params, bscfg[end+1:], nil
}

// Closing the Reprovider and Blockservice when the context is done.
func (p *Node) deferClose() {
	<-p.Ctx.Done()
	p.Reprovider.Close()
	p.Blockservice.Close()
}

// AddFile chunks and adds content to the DAGService from a reader. The content
// is stored as a UnixFS DAG (default for IPFS). It returns the root
// ipld.Node.
func (p *Node) AddPinFile(ctx context.Context, r io.Reader, params *AddParams) (ipld.Node, error) {

	bserv := blockservice.New(p.Blockstore, nil)
	dserv := merkledag.NewDAGService(bserv)

	if params == nil {
		params = &AddParams{}
	}
	if params.HashFun == "" {
		params.HashFun = "sha2-256"
	}

	prefix, err := merkledag.PrefixForCidVersion(1)
	if err != nil {
		return nil, fmt.Errorf("bad CID Version: %s", err)
	}

	prefix.MhType = uint64(multihash.SHA2_256)
	prefix.MhLength = -1

	dbp := helpers.DagBuilderParams{
		Dagserv:    dserv,
		RawLeaves:  params.RawLeaves,
		Maxlinks:   helpers.DefaultLinksPerBlock,
		NoCopy:     params.NoCopy,
		CidBuilder: &prefix,
	}

	spl := chunker.NewSizeSplitter(r, 1024*1024)
	dbh, err := dbp.New(spl)
	if err != nil {
		return nil, err
	}

	var n ipld.Node
	switch params.Layout {
	case "trickle":
		n, err = trickle.Layout(dbh)
	case "balanced", "":
		n, err = balanced.Layout(dbh)
	default:
		return nil, errors.New("invalid Layout")
	}
	return n, err
}

// GetFile returns a reader to a file as identified by its root CID. The file
// must have been added as a UnixFS DAG (default for IPFS).
func (p *Node) GetFile(ctx context.Context, c cid.Cid) (ufsio.ReadSeekCloser, error) {
	n, err := p.Get(ctx, c)
	if err != nil {
		return nil, err
	}
	return ufsio.NewDagReader(ctx, n, p)
}

// Getting the directory with the cid.
func (p *Node) GetDirectoryWithCid(ctx context.Context, c cid.Cid) (ufsio.Directory, error) {

	//links, _ :=
	//nodes := ipld.GetNodes(ctx, p, []cid.Cid{c})

	node, err := p.Get(ctx, c)

	if err != nil {
		return nil, err
	}
	directory, err := ufsio.NewDirectoryFromNode(p.DAGService, node)
	if err != nil {
		return nil, err
	}
	return directory, nil
}

// Getting the directory from the node.
func (p *Node) GetDirectory(ctx context.Context, c ipld.Node) (ufsio.Directory, error) {
	directory, err := ufsio.NewDirectoryFromNode(p.DAGService, c)
	if err != nil {
		return nil, err
	}
	return directory, nil
}

// Adding the directory of the pin to the path.
func (p *Node) AddPinDirectory(ctx context.Context, path string) (ipld.Node, error) {

	dirNode := ufsio.NewDirectory(p.DAGService)
	prefix, err := merkledag.PrefixForCidVersion(1)
	prefix.MhType = uint64(multihash.SHA2_256)

	dirNode.SetCidBuilder(cidutil.InlineBuilder{
		Builder: prefix,
	})

	err = filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				dirGetNode, _ := dirNode.GetNode()
				dirNode.AddChild(ctx, info.Name(), dirGetNode)
			} else {
				b, _ := os.ReadFile(path)
				fileNode, err := p.AddPinFile(ctx, bytes.NewReader(b), nil)
				dirNode.AddChild(ctx, info.Name(), fileNode)
				if err != nil {
					return err
				}
			}

			return nil
		})
	if err != nil {
		log.Println(err)
	}
	node, err := dirNode.GetNode()
	return node, nil
}
