// @Title  
// @Description  
// @Author  yang  2020/6/29 19:53
// @Update  yang  2020/6/29 19:53
package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
)

type Version struct {
	Version int      // 版本
	LastHeight int32   // 区块高度 (最近的一个区块)
	AddrFrom string  // Version 信息的发送方地址 (用于其他节点回复给谁消息)
}

// 已知节点 (初始时，设定为一些 稳定的节点地址（24小时不关机，稳定的服务节点）)
var knownNodes = []string{"localhost:3000"}
var nodeAddress string  // 本地节点地址

var blockInTransit [][]byte


const (
	NodeVersion = 0x00
	commandLength = 12
	CommandVersion = "version"
	CommandGetBlockS = "getblocks"
	CommandInv = "inv"
	CommandGetData = "getdata"
	CommandBlockData = "blockdata"
)

// 启动服务 (节点ID， 矿工比特币地址)
func StartServer(nodeID string) {
	nodeAddress = fmt.Sprintf("localhost:%s", nodeID)   // 例如：localhost:3001
	listener,err := net.Listen("tcp", nodeAddress)
	if err != nil {
		log.Panic(err)
	}

	defer listener.Close()

	var bc *BlockChain

	bc = InitBlockChain()

	// 单机模拟多个节点， 假设 端口3000 为稳定的服务节点（初始化创世区块）
	if nodeID == "3000"{
		wallets,err := NewWallets()
		var minerBTCAddress string
		if err != nil && os.IsNotExist(err) {
			minerBTCAddress = wallets.CreateWallet()
			gensisBlock := NewGensisBlock(minerBTCAddress)
			err := bc.AddBlock(gensisBlock)
			if err != nil {
				log.Panic("Add gensis Block failed , err : ", err)
			}
			// 更新UTXO SET
			set := &UTXOSet{bc}
			set.Reindex()
		}
	}

	if nodeAddress != knownNodes[0] {
		sendVersion(knownNodes[0], bc)
	}

	for {
		fmt.Println("for accept")
		conn, err := listener.Accept()
		fmt.Println("accept somethings")
		if err != nil {
			fmt.Printf("listener accept failed, err : %v\n", err)
			log.Panic(err)
		}

		fmt.Println("GO handleConnection")
		go handleConnection(conn, bc)
	}
}

// 处理监听到的 连接
func handleConnection(conn net.Conn, bc *BlockChain) {
	request,err := ioutil.ReadAll(conn)
	if err != nil {
		fmt.Printf("handleConnection Read failed! err : %v\n", err)
	}
	// 读取请求头数据，解析为 command
	command := bytesToCommand(request[:commandLength])
	fmt.Printf("command : %s\n", command)
	switch command {
	case CommandVersion:
		fmt.Printf("CommandVersion\n")
		handleVersion(request,bc)
	case CommandGetBlockS:
		fmt.Printf("CommandGetBlockS\n")
		handleGetBlocks(request,bc)
	case CommandInv:
		fmt.Printf("CommandInv\n")
		handleInv(request, bc)
	case CommandGetData:
		fmt.Printf("CommandGetData\n")
		handleGetData(request,bc)
	case CommandBlockData:
		fmt.Printf("CommandBlockData\n")
		handleBlockData(request, bc)
	default:
		break

	}
}
// 处理 CommandBlockData 命令请求
func handleBlockData(request []byte, bc *BlockChain) {
	var payload blockData
	err := gobDecode(request[commandLength:], &payload)
	if err != nil {
		fmt.Printf("handleBlockData decode failed! err : %v\n",err)
	}
	fmt.Printf("[blockdata] AddrFrom : %v, data %v\n", payload.AddrFrom,payload.Data)

	block := DeserializeBlock(payload.Data)
	err = bc.AddBlock(block)
	if err != nil {
		fmt.Printf("bc.AddBlock failed , err : %v\n", err)
	}

	// 从其他节点获取一个新的区块之后，判断 blockInTransit 还有没有未被获取的 区块信息, 有，则继续发送请求获取
	if len(blockInTransit) > 0 {
		blockHash := blockInTransit[0]
		sendGetData(payload.AddrFrom,"block", blockHash)
		updateBlockInTransit(blockHash)
	}else {   // 当区块更新完毕，则更新 UTXO SET
		set := UTXOSet{bc}
		set.Reindex()
	}

	fmt.Printf("Receive a new Block\n")

}
// 处理 CommandGetData 命令请求
func handleGetData(request []byte, bc *BlockChain) {
	var payload getData
	err := gobDecode(request[commandLength:], &payload)
	if err != nil {
		fmt.Printf("handleGetData decode failed! err : %v\n",err)
	}
	fmt.Printf("[inv] AddrFrom : %v, kind : %v, ID %v\n", payload.AddrFrom,payload.Kind,payload.ID)

	if payload.Kind == "block" {
		block, err := bc.GetBlockByHashID(payload.ID)
		if err != nil {
			fmt.Printf("bc.getBlock failed , err : %v\n", err)
		}
		fmt.Printf("sendBlock, height : %d\n", block.Height)
		sendBlock(payload.AddrFrom, &block)
	}
}

type blockData struct {
	AddrFrom string
	Data []byte
}

func sendBlock(addrTo string, block *Block) {
	v := blockData{
		AddrFrom: nodeAddress,
		Data:     block.Serialize(),
	}

	payload := gobEncode(v)
	request := append(commandToBytes(CommandBlockData), payload...)
	fmt.Printf("sendBlock : data %v\n", v.Data)
	sendData(addrTo, request)
}
// 处理 CommandInv 命令请求
func handleInv(request []byte, bc *BlockChain) {
	var payload inv
	err := gobDecode(request[commandLength:], &payload)
	if err != nil {
		fmt.Printf("handleInv decode failed! err : %v\n",err)
	}
	fmt.Printf("[inv] AddrFrom : %v, kind : %v, item : %v\n", payload.AddrFrom,payload.Kind,payload.Items)

	if payload.Kind == "block" {
		blockInTransit = payload.Items
		blockHash := payload.Items[0]
		sendGetData(payload.AddrFrom,"block", blockHash)

		updateBlockInTransit(blockHash)
	}


}
// 更新 blockInTransit  (将以发送的 block hash 从列表中剔除)
func updateBlockInTransit(alreadySendHash []byte) {
	var newBlockInTransit  [][]byte
	for _,b := range blockInTransit{
		if bytes.Compare(b, alreadySendHash) != 0 {
			newBlockInTransit = append(newBlockInTransit, b)
		}
	}
	blockInTransit = newBlockInTransit
}

type getData struct {
	AddrFrom string
	Kind string
	ID []byte
}
func sendGetData(addrTo string, kind string, hash []byte) {
	v := getData{
		AddrFrom: nodeAddress,
		Kind:     kind,
		ID:       hash,
	}
	payload := gobEncode(v)
	request := append(commandToBytes(CommandGetData), payload...)

	sendData(addrTo, request)
}
// 处理 CommandGetBlock 命令请求
func handleGetBlocks(request []byte, bc *BlockChain) {
	var payload getBlock
	err := gobDecode(request[commandLength:], &payload)
	if err != nil {
		fmt.Printf("handleGetBlock decode failed! err : %v\n",err)
	}
	fmt.Printf("[getBlock] AddrFrom : %v\n", payload.AddrFrom)

	blockHashs := bc.GetBlockHashs()
	sendInv(payload.AddrFrom,"block",blockHashs)
}

type inv struct {
	AddrFrom string
	Kind string
	Items [][]byte
}
// 发送所有区块hash 信息
func sendInv(addrTo string, kind string, blockHashs [][]byte) {
	v := inv{
		AddrFrom: nodeAddress,
		Kind:     kind,
		Items:    blockHashs,
	}

	payload := gobEncode(v)
	request := append(commandToBytes(CommandInv), payload...)

	sendData(addrTo, request)
}
// 处理 CommandVersion 命令请求
func handleVersion(request []byte, bc *BlockChain) {
	var payload Version
	err := gobDecode(request[commandLength:], &payload)
	if err != nil {
		fmt.Printf("handleVersion decode failed! err : %v\n",err)
	}
	fmt.Printf("[Version] version : %v,lastHeight : %v, addr : %v\n", payload.Version,payload.LastHeight,payload.AddrFrom)
	// 本地最新区块高度
	myLastHeight := bc.getLashHeight()
	// 外部节点最新区块高度
	foreignerLastHeight := payload.LastHeight
	// 如果 myLastHeight < foreignerLastHeight 成立，则代表 外部节点区块信息比本地多
	// 所以要发送 getblock 请求
	if myLastHeight < foreignerLastHeight {
		sendGetBlocks(payload.AddrFrom,bc)

		fmt.Printf("send getblock request\n")

	}else {  // 否则，发送本地的版本信息 作为回复 (本地的版本号，本地节点地址，本地最新区块高度)
		sendVersion(payload.AddrFrom, bc)
	}

	// 如果 外部节点不存在于 已知节点列表中时，则将其 添加到 knownNodes
	if !nodeIsKnown(payload.AddrFrom) {
		knownNodes = append(knownNodes, payload.AddrFrom)
	}

}
// 判断 from 节点地址是否存在 已知列表中 knownNodes
func nodeIsKnown(from string) bool{
	for _,node := range knownNodes {
		if node == from {
			return true
		}
	}
	return false
}

type getBlock struct {
	AddrFrom string
}
// 发送 获取区块的 请求
func sendGetBlocks(addrTo string, bc *BlockChain) {

	v := getBlock{AddrFrom:nodeAddress }
	payload := gobEncode(v)
	request := append(commandToBytes(CommandGetBlockS), payload...)
	sendData(addrTo,request)
}

// 发送本地区块链的 版本信息 (版本号,最新区块高度，本地节点地址)
func sendVersion(addrTo string, bc *BlockChain) {

	lastHeight := bc.getLashHeight()
	fmt.Printf("sendVersion lastHeight : %d\n", lastHeight)
	v := Version{
		Version:    NodeVersion,
		LastHeight: lastHeight,
		AddrFrom:   nodeAddress,
	}

	data := gobEncode(v)
	request := append(commandToBytes(CommandVersion), data...)
	sendData(addrTo, request)
}
// 发送数据给其他节点
func sendData(addrTo string, request []byte) {
	conn, err := net.Dial("tcp", addrTo)
	if err != nil {
		fmt.Printf("node : %s is not available!\n", addrTo)

		updateKnownNodes(addrTo)

		/*
				更新完毕之后呢？  是否要尝试连接其他节点
		 */
	}
	defer  conn.Close()

	_, err = io.Copy(conn,bytes.NewReader(request))
	if err != nil {
		fmt.Printf("sendData failed!\n")
	}
	fmt.Printf("send data success!")
}

// 当某个节点连接不上，则更新 knownNodes , 剔除该无效节点
func updateKnownNodes(unReachNodeAddr string) {
	var updateNodes []string
	for _, node := range knownNodes {
		if node != unReachNodeAddr {
			updateNodes = append(updateNodes,node)
		}
	}
	knownNodes = updateNodes
}

// 命令字符串转换为 字节数组  (便于网络传输)
func commandToBytes(command string) []byte {
	var bytes [commandLength]byte
	for i,c := range command {
		bytes[i] = byte(c)
	}
	return bytes[:]
}
// 字节数组转换为 命令字符串 (用于判断是哪种命令)
func bytesToCommand(bytes []byte) string {
	var command []byte
	for _,b := range bytes {
		if b != 0x00 {
			command = append(command, b)
		}
	}
	return string(command)
}

// gob 编码序列化
func gobEncode(v interface{}) []byte {
	var buf bytes.Buffer
	encode := gob.NewEncoder(&buf)
	err := encode.Encode(v)
	if err != nil {
		log.Panic(err)
	}
	return buf.Bytes()
}

// gob 解码 (从网络中获取的数据，解析为 结构体对象)
func gobDecode(data []byte, v interface{}) error{
	var buf bytes.Buffer
	buf.Write(data)
	decode := gob.NewDecoder(&buf)
	err := decode.Decode(v)
	if err != nil {
		return err
	}
	return nil
}
