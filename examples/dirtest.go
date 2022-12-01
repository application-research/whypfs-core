package main

import (
	"context"
	whypfs "github.com/application-research/whypfs-core"
	"github.com/ipfs/go-cid"
)

// Creating a new whypfs node, bootstrapping it with the default bootstrap peers, adding a file to the whypfs network, and
// then retrieving the file from the whypfs network.
func DirTest() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	whypfsPeer, err := whypfs.NewNode(whypfs.NewNodeParams{
		Ctx:       ctx,
		Datastore: whypfs.NewInMemoryDatastore(),
	})

	whypfsPeer.BootstrapPeers(whypfs.DefaultBootstrapPeers())

	c1, _ := cid.Decode("bafybeigvgzoolc3drupxhlevdp2ugqcrbcsqfmcek2zxiw5wctk3xjpjwy")
	rsc1, err := whypfsPeer.GetDirectoryWithCid(ctx, c1)
	if err != nil {
		panic(err)
	}
	rsc1.GetNode()
}
