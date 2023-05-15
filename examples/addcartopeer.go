package main

import (
	"bytes"
	"context"
	"fmt"
	whypfs "github.com/application-research/whypfs-core"
	"github.com/ipfs/go-cid"
	chunker "github.com/ipfs/go-ipfs-chunker"
	format "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-unixfs/importer/balanced"
	"github.com/ipfs/go-unixfs/importer/helpers"
	"github.com/ipld/go-car"
	"github.com/multiformats/go-multihash"
	"io/ioutil"
	"net/http"
)

func GetPublicIP() (string, error) {
	resp, err := http.Get("https://ifconfig.me") // important to get the public ip if possible.
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func GetCidBuilderDefault() cid.Builder {
	cidBuilder, err := merkledag.PrefixForCidVersion(1)
	if err != nil {
		panic(err)
	}
	cidBuilder.MhType = uint64(multihash.SHA2_256)
	cidBuilder.MhLength = -1
	return cidBuilder
}
func CreateNodeRaw(data []byte, whypfsPeer whypfs.Node) format.Node {

	//rootNode := &merkledag.ProtoNode{}
	cidBuilder, err := merkledag.PrefixForCidVersion(1)
	cidBuilder.MhType = uint64(multihash.SHA2_256)
	cidBuilder.MhLength = -1

	dbp := helpers.DagBuilderParams{
		Dagserv:    whypfsPeer.DAGService,
		RawLeaves:  true,
		Maxlinks:   helpers.DefaultLinksPerBlock,
		NoCopy:     false,
		CidBuilder: &cidBuilder,
	}
	//
	spl := chunker.NewSizeSplitter(bytes.NewReader(data), 1024*1024)
	dbh, err := dbp.New(spl)
	if err != nil {
		panic(err)
	}
	node1Raw1, err := balanced.Layout(dbh)

	if err != nil {
		panic(err)
	}

	return node1Raw1
}

// Creating a new whypfs node, bootstrapping it with the default bootstrap peers, adding a file to the whypfs network, and
// then retrieving the file from the whypfs network.
func AddCarToPeer() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publicIp, err := GetPublicIP()
	newConfig := &whypfs.Config{
		ListenAddrs: []string{
			"/ip4/0.0.0.0/tcp/6745",
			"/ip4/" + publicIp + "/tcp/6745",
		},
		AnnounceAddrs: []string{
			"/ip4/0.0.0.0/tcp/6745",
			"/ip4/" + publicIp + "/tcp/6745",
		},
	}
	params := whypfs.NewNodeParams{
		Ctx:       context.Background(),
		Datastore: whypfs.NewInMemoryDatastore(),
		Repo:      ".whypfs1",
	}

	// node
	params.Config = params.ConfigurationBuilder(newConfig)
	whypfsPeer, err := whypfs.NewNode(params)
	whypfsPeer.BootstrapPeers(whypfs.DefaultBootstrapPeers())

	//node1Raw := merkledag.NewRawNode([]byte("letsrebuildtolearnnewthings!1"))
	node1Raw := CreateNodeRaw([]byte("letsrebuildtolearnnewthings!1"), *whypfsPeer)
	node2Raw := CreateNodeRaw([]byte("letsrebuildtolearnnewthings!2"), *whypfsPeer)
	node3Raw := CreateNodeRaw([]byte("letsrebuildtolearnnewthings!3"), *whypfsPeer)
	node4Raw := CreateNodeRaw([]byte("letsrebuildtolearnnewthings!4"), *whypfsPeer)

	node1 := &merkledag.ProtoNode{}
	node1.AddNodeLink("node1", node1Raw)
	node1.SetCidBuilder(GetCidBuilderDefault())

	node2 := &merkledag.ProtoNode{}
	node2.AddNodeLink("first", node1)
	node2.AddNodeLink("node2", node2Raw)
	node2.SetCidBuilder(GetCidBuilderDefault())

	node3 := &merkledag.ProtoNode{}
	node3.AddNodeLink("node3", node2)
	node3.AddNodeLink("node31", node3Raw)
	node3.SetCidBuilder(GetCidBuilderDefault())

	node4 := &merkledag.ProtoNode{}
	//node4.AddNodeLink("node4 - wow node1", node1)
	//node4.AddNodeLink("node4 - wow node2", node2)
	node4.AddNodeLink("node43", node3)

	// file
	file, err := ioutil.ReadFile("test_file_for_car1.log")
	node4.AddNodeLink("node4root", node4Raw)
	node4.SetData(file)
	node4.SetCidBuilder(GetCidBuilderDefault())

	//rootNode.AddNodeLink("root - alright", node4)
	//rootNode.SetCidBuilder(cidBuilder)
	//rootNode.SetData([]byte("root - alright"))
	//fmt.Println("Root CID before: ", rootNode.Cid().String())
	assertAddNodes(*whypfsPeer, node1Raw, node2Raw, node3Raw, node4Raw, node1, node2, node3, node4)

	buf := new(bytes.Buffer)
	if err := car.WriteCar(context.Background(), whypfsPeer.DAGService, []cid.Cid{node4.Cid()}, buf); err != nil {
		panic(err)
	}
	fmt.Println("CAR file size: ", buf.Len())
	ch, err := car.LoadCar(context.Background(), whypfsPeer.Blockservice.Blockstore(), buf)
	if err != nil {
		panic(err)
	}

	fmt.Println("Root CID LoadedCAR: ", ch.Roots[0].String())
	rootCidToWrite, err := whypfsPeer.Get(ctx, ch.Roots[0])
	// write raw to data file
	raw := rootCidToWrite.RawData()
	err = ioutil.WriteFile("data_file_for_car1.car", raw, 0644)

	for _, c := range ch.Roots {
		rootCid, err := whypfsPeer.Get(ctx, c)
		fmt.Println("Root CID after: ", rootCid.String())

		if err != nil {
			panic(err)
		}
		traverseLinks(ctx, whypfsPeer.DAGService, rootCid)
	}

	for _, nd := range []format.Node{node1Raw, node2Raw, node3Raw, node4Raw, node1, node2, node3, node4} {
		has, err := whypfsPeer.Blockstore.Has(context.TODO(), nd.Cid())
		if err != nil {
			panic(err)
		}

		if !has {
			fmt.Println("Node not found in blockstore: ", nd.Cid().String())
			panic("Node not found in blockstore")
		}
		fmt.Println("Node found in blockstore: ", nd.Cid().String())
	}
}

func assertAddNodes(ds whypfs.Node, nds ...format.Node) {
	for _, nd := range nds {
		fmt.Println("Adding node: ", nd.Cid().String())
		err := ds.DAGService.Add(context.Background(), nd)
		if err != nil {
			fmt.Println("Error adding node: ", err)
		}
		//fmt.Println("Added node: ", addedNode.Cid().String())
	}
}

// function to traverse all links
func traverseLinks(ctx context.Context, ds format.DAGService, nd format.Node) {
	for _, link := range nd.Links() {
		node, err := link.GetNode(ctx, ds)
		if err != nil {
			panic(err)
		}
		fmt.Println("Node CID: ", node.Cid().String())
		traverseLinks(ctx, ds, node)
	}
}
