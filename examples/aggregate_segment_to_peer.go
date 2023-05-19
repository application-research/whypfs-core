package main

import (
	"bytes"
	"context"
	whypfs "github.com/application-research/whypfs-core"
	"github.com/ipfs/go-merkledag"
	"github.com/multiformats/go-multihash"
	"io"
	"os"
)

// Creating a new whypfs node, bootstrapping it with the default bootstrap peers, adding a file to the whypfs network, and
// then retrieving the file from the whypfs network.
func main() {
	_, cancel := context.WithCancel(context.Background())
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

	mainBuf := new(bytes.Buffer)
	//mainFile := bytes.NewReader(buf.Bytes())

	p1, err := os.Open("test_file_for_car1.log")
	if err != nil {
		panic(err)
	}
	io.Copy(mainBuf, p1)

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
