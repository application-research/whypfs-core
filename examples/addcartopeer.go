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
)

// Creating a new whypfs node, bootstrapping it with the default bootstrap peers, adding a file to the whypfs network, and
// then retrieving the file from the whypfs network.
func AddCarToPeer() {
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
	rootNode := &merkledag.ProtoNode{}
	node1.AddNodeLink("node1 - yehey", baseNode)
	node1.SetData([]byte("node1 - yehey"))
	node2.AddNodeLink("node2  - nice", node1)
	node2.SetData([]byte("node2  - nice"))
	node3.AddNodeLink("node3 - wow", node2)
	node3.SetData([]byte("node3 - wow"))
	rootNode.AddNodeLink("root - alright", node3)
	rootNode.SetData([]byte("root - alright"))

	fmt.Println("Root CID before: ", rootNode.Cid().String())

	assertAddNodes(whypfsPeer.DAGService, rootNode, node3, node2, node1, baseNode)

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
	fmt.Println(string(ch.Version))

	rootCid, err := whypfsPeer.Get(ctx, rootNode.Cid())
	if err != nil {
		panic(err)
	}
	fmt.Println("Root CID after: ", rootCid.String())

	for _, link := range rootCid.Links() {
		fmt.Println(link.Cid.String())
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
