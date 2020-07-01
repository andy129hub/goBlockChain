// @Title  
// @Description  
// @Author  yang  2020/6/23 14:11
// @Update  yang  2020/6/23 14:11
package main

import (
	"encoding/hex"
	"fmt"
)

// 测试 命令行控制
func TestCLI() {
	// blockChain := InitBlockChain(minerAddress)
	cli := CLI{bc:nil}
	cli.Run()
}

func TestAddress() {

	wallet := NewWallet()

	fmt.Printf("私钥：%x\n", wallet.PrivateKey.D.Bytes())
	fmt.Printf("公钥: %x\n", wallet.PublicKey)
	fmt.Printf("地址：%s\n", wallet.GetAddress())

	fmt.Printf("地址是否有效：%t\n", ValidateAddress(wallet.GetAddress()))


	fmt.Printf("18rNG8sdeyLDoa88896vMnbXkEkLCsVnYj 地址是否有效：%t\n", ValidateAddress([]byte("18rNG8sdeyLDoa88896vMnbXkEkLCsVnYj")))
}

// 测试 bolt db
func TestBoltDB(){
	blockChain := InitBlockChainByTest(minerAddress)
	blockChain.MinerBlock([]*Transaction{})
	blockChain.MinerBlock([]*Transaction{})

	blockChain.printBlockChain()
}


// 测试序列化与反序列化
func TestNewSerialize() {
	header := BlockHeader{
		Version:    2,
		PrevHash:   []byte{},
		MerkleRoot: []byte{},
		Time:       1231643460,
		Bits:       486604799,
		Nonce:      0,
	}

	block := Block{
		Height:0,
		Hash:         []byte{},   // 区块头hash
		Header:       &header,    // 区块头
		Transactions: []*Transaction{}, // 交易
	}

	deBlock := block.Serialize()

	b := DeserializeBlock(deBlock)

	fmt.Println(b)
}


// 测试 POW
func testPOW(){
	var version int32 = 2
	// 上一个区块的 hash  ---> 链上前一个区块的散列值的参考值
	prevHash, _ := hex.DecodeString("00000000839a8e6886ab5951d76f411475428afc90947ee320161bbf18eb6048")
	ReverseBytes(prevHash)

	// 交易 : 将每一笔 Transaction 加入到 transactions 集合中
	newTX := NewCoinbaseTX(minerAddress, genesisData)
	ReverseBytes(newTX.ID)
	fmt.Printf("newTX.ID：%x\n", newTX.ID)   // d41e1b23ba0377ac2f2c9e2ffe94a820ad04099c0e40e8f780ff1270891c6595

	transactions := []*Transaction{newTX}
	datas := [][]byte {
		newTX.ID,
	}

	// 默克尔根 hash : 根据每一笔交易 的 txid hash 构建默克尔树，计算出 默克尔根 hash
	merkleRoot := NewMerkleTree(datas)

	// 大小端转换
	ReverseBytes(merkleRoot.RootNode.Data)
	merkleRootHash := merkleRoot.RootNode.Data

	fmt.Printf("默克尔根：%x\n", merkleRootHash)  // 结果： 95651c897012ff80f7e8400e9c0904ad20a894fe2f9e2c2fac7703ba231b1ed4

	// 时间戳
	var timeInt int32 = 1231643460
	//fmt.Println(timeInt)

	//fmt.Printf("time : %x\n", IntToHex(timeInt))
	// 目标难度值  ---> 当前区块 POW 算法的目标难度值
	var bits int32 = 486604799    // 调低难度值
	//fmt.Printf("bits : %x\n", IntToHex(bits))

	// 根据 版本号，上一区块hash, 默克尔根, 时间戳，目标难度值，计算求出 区块的随机数 nonce 从而得到正确的 区块头 hash, 并返回 Block 对象。
	minerWork(version, prevHash,merkleRootHash,transactions, timeInt,bits)
	//fmt.Println("--------------------------挖矿成功，出块儿！------------------------")
	//fmt.Printf("区块信息： %x\n", block.Hash)
}

// 挖矿
func minerWork(version int32,prevHash,merkleRoot []byte,transaction []*Transaction, _time,bits int32){

	// 初始化 区块头信息  (版本，前一区块hash, 默克尔根, 时间戳, 目标难度值，随机数)
	header := BlockHeader{
		Version:    version,
		PrevHash:   prevHash,
		MerkleRoot: merkleRoot,
		Time:       _time,
		Bits:       bits,
		Nonce:      0,
	}

	block := Block{
		Height:0,
		Hash:         []byte{},   // 区块头hash
		Header:       &header,    // 区块头
		Transactions: transaction, // 交易
	}

	pow := NewPOW(&block)

	// 尝试使用 goroutine 多线程进行挖矿
	// pow.startWork()

	// 测试...
	pow.Run2()

}