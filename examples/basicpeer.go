package main

import (
	"bytes"
	"context"
	"fmt"
	whypfs "github.com/application-research/whypfs-core"
	"io"

	"github.com/ipfs/go-cid"
)

// Creating a new whypfs node, bootstrapping it with the default bootstrap peers, adding a file to the whypfs network, and
// then retrieving the file from the whypfs network.
func basicPeer() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	whypfsPeer, err := whypfs.NewNode(whypfs.NewNodeParams{
		Ctx:       ctx,
		Datastore: whypfs.NewInMemoryDatastore(),
	})
	whypfsPeer.BootstrapPeers(whypfs.DefaultBootstrapPeers())

	file, err := whypfsPeer.AddPinFile(context.Background(), bytes.NewReader([]byte("letsrebuildtolearnnewthings!")), nil)
	if err != nil {
		return
	}

	fmt.Println("File CID: ", file.Cid().String())

	c, _ := cid.Decode("bafybeiawc5enlmxtwdbnts3mragh5eyhl3wn5qekvimw72igdj45lixbo4")
	rsc, err := whypfsPeer.GetFile(ctx, c)
	if err != nil {
		panic(err)
	}
	defer rsc.Close()
	content, err := io.ReadAll(rsc)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(content))
}
