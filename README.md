# WhyPFS Core

![go build](https://github.com/application-research/whypfs-core/actions/workflows/go.yml/badge.svg)
![code ql](https://github.com/application-research/whypfs-core/actions/workflows/codeql.yml/badge.svg)

A lightweight importable library to run a WhyPFS node.

Easiest way to setup an WhyPFS / Estuary core node that peers with any nodes within the IPFS network. This core library decouples the
node initialization setup based on the Why's https://github.com/whyrusleeping/whypfs

# Features
- Significantly improved default Bitswap parameters
- Default usage of the accelerated DHT client
- Default flatfs parameters work well at large scales on good filesystems (ext4, xfs, probably others)


# Setup a new WhyPFS node
Using the default will give the most optimal configuration based on whyPFS. It'll also use the default
boostrap nodes.

```shell
go get github.com/application-research/whypfs-core
```

## Setup a node
```
peer, err := NewNode(NewNodeParams{Ctx: context.Background()})	
if err != nil {
    t.Fatal(err)
}
```

## Set up a node with your own config
```
// initialize a node parameter
params := NewNodeParams{
    Ctx:       context.Background(),
    Datastore: NewInMemoryDatastore(),
}

// create a new config of your own
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

// set it
params.Config = params.ConfigurationBuilder(newConfig)
myNode, err := NewNode(params)
if err1 != nil {
    t.Fatal(err)
}
```

## Add/Pin and Get a file
```
node, err := peer.AddPinFile(context.Background(), bytes.NewReader([]byte("letsrebuildtolearnnewthings!")), nil)
content, err := peer.GetFile(context.Background(), node.Cid())
```

## Add/Pin Directory and Get the ipld.Node of the directory
```
node, err := peer.AddPinDirectory(context.Background(), "./test/test_directory")
mainDirNode, err := peer.GetDirectory(context.Background(), node)
```

## Provides
- An ipld.DAGService.
- AddPinFile method to add a file to a node
- AddPinDirectory function to add a directory to a node
- GetFile function to get a file using a CID
- GetDirectory function to retrieve an entire directory from a ipld.Node
- Custom bootstrap nodes

## Examples
There are a few examples on how to utilize the node which includes
- how to create a running peer node.
- how to create a CAR file and add it to the peer node
- how to add a file / dir to the peer node.