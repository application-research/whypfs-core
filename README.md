# WhyPFS Core
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
p1, err := p1, err1 := NewNode(NewNodeParams{Ctx: context.Background()})	
if err != nil {
    t.Fatal(err)
}
```

## Add/Pin and Get a file
```
node, err := p1.AddPinFile(context.Background(), bytes.NewReader([]byte("letsrebuildtolearnnewthings!")), nil)
content, err := p1.GetFile(context.Background(), node.Cid())
```

## Add/Pin Directory and Get the ipld.Node of the directory
```
node, err := p1.AddPinDirectory(context.Background(), "./test/test_directory")
mainDirNode, err := p1.GetDirectory(context.Background(), node)
```

## Provides
- An ipld.DAGService.
- AddPinFile method to add a file to a node
- AddPinDirectory function to add a directory to a node
- GetFile function to get a file using a CID
- GetDirectory function to retrieve an entire directory from a ipld.Node
- Custom bootstrap nodes