package main

import (
	"context"
	"fmt"
	whypfs "github.com/application-research/whypfs-core"
	"io"

	"github.com/ipfs/go-cid"
)

// Creating a new whypfs node, bootstrapping it with the default bootstrap peers, adding a file to the whypfs network, and
// then retrieving the file from the whypfs network.
func arTest() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	whypfsPeer, err := whypfs.NewNode(whypfs.NewNodeParams{
		Ctx:       ctx,
		Datastore: whypfs.NewInMemoryDatastore(),
	})

	whypfsPeer.BootstrapPeers(whypfs.DefaultBootstrapPeers())

	c1, _ := cid.Decode("QmbJGGJkjGfYCmHqwoMjLTbUA6bdcFBbNdWChFY6dKNRWx")
	rsc1, err := whypfsPeer.GetFile(ctx, c1)
	if err != nil {
		panic(err)
	}
	defer rsc1.Close()
	content1, err := io.ReadAll(rsc1)
	if err != nil {
		panic(err)
	}

	c2, _ := cid.Decode("QmPecv5jpAPVG3LUN8fbptnaPHFzt3SRtxC8e1XZCVMiSM")
	rsc2, err := whypfsPeer.GetFile(ctx, c2)
	if err != nil {
		panic(err)
	}
	defer rsc2.Close()
	content2, err := io.ReadAll(rsc2)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(content1))
	fmt.Println(string(content2))
}
