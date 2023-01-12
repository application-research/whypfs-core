package main

import (
	"bytes"
	"context"
	"fmt"
	whypfs "github.com/application-research/whypfs-core"
	"github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
	"github.com/ipld/go-car"
	"io/ioutil"
)

// Creating a new whypfs node, bootstrapping it with the default bootstrap peers, adding a file to the whypfs network, and
// then retrieving the file from the whypfs network.
func AddCarToPeerV1() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	whypfsPeer, err := whypfs.NewNode(whypfs.NewNodeParams{
		Ctx:       ctx,
		Datastore: whypfs.NewInMemoryDatastore(),
	})
	whypfsPeer.BootstrapPeers(whypfs.DefaultBootstrapPeers())

	baseNode := merkledag.NewRawNode([]byte("letsrebuildtolearnnewthings!"))
	node1 := &merkledag.ProtoNode{}
	node2 := &merkledag.ProtoNode{}
	node3 := &merkledag.ProtoNode{}
	node4 := &merkledag.ProtoNode{}
	rootNode := &merkledag.ProtoNode{}
	node1.AddNodeLink("node1 - yehey", baseNode)
	node1.SetData([]byte("node1 - yehey"))
	node2.AddNodeLink("node2  - nice", node1)
	node2.SetData([]byte("node2  - nice"))
	node3.AddNodeLink("node3 - wow", node2)
	node3.SetData([]byte("node3 - wow"))
	node4.AddNodeLink("node4 - cool", node3)

	// file
	file, err := ioutil.ReadFile("test_file_for_car1.log")
	node4.SetData(file)

	rootNode.AddNodeLink("root - alright", node4)
	rootNode.SetData([]byte("root - alright"))
	go fmt.Println("Root CID before: ", rootNode.Cid().String())

	assertAddNodes(whypfsPeer.DAGService, rootNode, node4, node3, node2, node1, baseNode)

	buf := new(bytes.Buffer)
	if err := car.WriteCar(context.Background(), whypfsPeer.DAGService, []cid.Cid{rootNode.Cid()}, buf); err != nil {
		panic(err)
	}
	fmt.Println("CAR file size: ", buf.Len())
	ch, err := car.LoadCar(context.Background(), whypfsPeer.Blockservice.Blockstore(), buf)
	if err != nil {
		panic(err)
	}

	fmt.Print(rootNode.Cid().String())
	for _, c := range ch.Roots {
		rootCid, err := whypfsPeer.Get(ctx, c)
		fmt.Println("Root CID after: ", rootCid.String())
		if err != nil {
			panic(err)
		}
		traverseLinks(ctx, whypfsPeer.DAGService, rootCid)
	}
}

func assertAddNodes(ds format.DAGService, nds ...format.Node) {
	for _, nd := range nds {
		fmt.Println("Adding node: ", nd.Cid().String())
		if err := ds.Add(context.Background(), nd); err != nil {
			panic(err)
		}
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
