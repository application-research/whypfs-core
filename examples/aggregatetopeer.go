package main

import (
	"context"
	"fmt"
	whypfs "github.com/application-research/whypfs-core"
	"github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-unixfs/io"
	"github.com/multiformats/go-multihash"
	"io/ioutil"
	"os"
)

// Creating a new whypfs node, bootstrapping it with the default bootstrap peers, adding a file to the whypfs network, and
// then retrieving the file from the whypfs network.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//publicIp, err := GetPublicIP()
	newConfig := &whypfs.Config{
		ListenAddrs: []string{
			"/ip4/0.0.0.0/tcp/6745",
			//"/ip4/" + publicIp + "/tcp/6745",
		},
		AnnounceAddrs: []string{
			"/ip4/0.0.0.0/tcp/6745",
			//"/ip4/" + publicIp + "/tcp/6745",
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

	//f, _ := os.Open("data_file_for_car1.car")
	cidBuilder, err := merkledag.PrefixForCidVersion(1)
	if err != nil {
		panic(err)
	}
	cidBuilder.MhType = uint64(multihash.SHA2_256)
	cidBuilder.MhLength = -1

	dserv := merkledag.NewDAGService(whypfsPeer.Blockservice)
	dir := io.NewDirectory(dserv)
	node1Raw := merkledag.NewRawNode([]byte("aaaa"))
	node2Raw := merkledag.NewRawNode([]byte("bbbb"))
	node3Raw := merkledag.NewRawNode([]byte("cccc"))
	fileForNode4, err := ioutil.ReadFile("test_file_for_car1.log")
	if err != nil {
		panic(err)
	}
	node4Raw := merkledag.NewRawNode(fileForNode4)

	fileForNode5, err := ioutil.ReadFile("test_file_for_car2.log")
	if err != nil {
		panic(err)
	}
	node5Raw := merkledag.NewRawNode(fileForNode5)

	fileForNode6, err := ioutil.ReadFile("test_file_for_car3.log")
	if err != nil {
		panic(err)
	}
	node6Raw := merkledag.NewRawNode(fileForNode6)

	fileForNode7, err := ioutil.ReadFile("test_file_for_car4.log")
	if err != nil {
		panic(err)
	}
	node7Raw := merkledag.NewRawNode(fileForNode7)

	//err = dir.AddChild(ctx, fmt.Sprintf("%d-%s", node1Raw.ID, c.Name), nd)
	err = dir.AddChild(ctx, fmt.Sprintf("%d-%s", node1Raw.Cid()), node1Raw)
	err = dir.AddChild(ctx, fmt.Sprintf("%d-%s", node2Raw.Cid()), node2Raw)
	err = dir.AddChild(ctx, fmt.Sprintf("%d-%s", node3Raw.Cid()), node3Raw)
	err = dir.AddChild(ctx, fmt.Sprintf("%d-%s", node4Raw.Cid()), node4Raw)
	err = dir.AddChild(ctx, fmt.Sprintf("%d-%s", node5Raw.Cid()), node5Raw)
	err = dir.AddChild(ctx, fmt.Sprintf("%d-%s", node6Raw.Cid()), node6Raw)
	err = dir.AddChild(ctx, fmt.Sprintf("%d-%s", node7Raw.Cid()), node7Raw)

	dirNd, err := dir.GetNode()
	dir.SetCidBuilder(cidBuilder)
	if err != nil {
		panic(err)
	}

	whypfsPeer.DAGService.Add(ctx, node1Raw)
	whypfsPeer.DAGService.Add(ctx, node2Raw)
	whypfsPeer.DAGService.Add(ctx, node3Raw)
	whypfsPeer.DAGService.Add(ctx, node4Raw)
	whypfsPeer.DAGService.Add(ctx, node5Raw)
	whypfsPeer.DAGService.Add(ctx, node6Raw)
	whypfsPeer.DAGService.Add(ctx, node7Raw)
	whypfsPeer.DAGService.Add(ctx, dirNd)

	fmt.Println("dirNd.Cid()", dirNd.Cid())

	for _, v := range dirNd.Links() {
		fmt.Println("v.Cid", v.Cid)

	}

	// show raw data
	dirNdRaw, err := whypfsPeer.DAGService.Get(ctx, dirNd.Cid())
	if err != nil {
		panic(err)
	}

	writeToFile, err := os.OpenFile("test_file_for_car_agg.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	writeToFile.Write(dirNdRaw.RawData())
	writeToFile.Close()

	for _, v := range dirNdRaw.Links() {
		fmt.Println("v.Cid", v.Cid)
		// get node
		lNd, err := v.GetNode(context.Background(), whypfsPeer.DAGService)
		if err != nil {
			panic(err)
		}
		fmt.Println("v.RawData", string(lNd.RawData()))
	}

}

//func fastCommp(reader io.ReadSeekCloser) (writer.DataCIDSize, error) {
//	// Check if the file is a CARv2 file
//	isVarV2, headerInfo := checkCarV2(reader)
//	var streamBuf *bufio.Reader
//	cp := new(commp.Calc)
//	if isVarV2 {
//		// Extract the CARv1 data from the CARv2 file
//		sliced, err := extractCarV1(reader, int(headerInfo.DataOffset), int(headerInfo.DataSize))
//		if err != nil {
//			panic(err)
//		}
//		streamBuf = bufio.NewReaderSize(
//			io.TeeReader(sliced, cp),
//			BufSize,
//		)
//	} else {
//		// Read the file as a CARv1 file
//		streamBuf = bufio.NewReaderSize(
//			io.TeeReader(reader, cp),
//			BufSize,
//		)
//	}
//
//	// The length of the stream
//	var streamLen int64
//
//	// Read the header
//	carHeader, streamLen, err := readCarHeader(streamBuf, streamLen)
//	if err != nil {
//		return writer.DataCIDSize{}, err
//	}
//
//	if carHeader.Version == 1 || carHeader.Version == 2 {
//		streamLen, err = process(streamBuf, streamLen)
//		if err != nil {
//			log.Fatal(err)
//			return writer.DataCIDSize{}, err
//		}
//	} else {
//		return writer.DataCIDSize{}, fmt.Errorf("invalid car version: %d", carHeader.Version)
//	}
//
//	n, err := io.Copy(io.Discard, streamBuf)
//	streamLen += n
//	if err != nil && err != io.EOF {
//		log.Fatalf("unexpected error at offset %d: %s", streamLen, err)
//	}
//
//	rawCommP, paddedSize, err := cp.Digest()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	commCid, err := commcid.DataCommitmentV1ToCID(rawCommP)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	return writer.DataCIDSize{
//		PayloadSize: streamLen,
//		PieceSize:   abi.PaddedPieceSize(paddedSize),
//		PieceCID:    commCid,
//	}, nil
//}
