// @Title  
// @Description  
// @Author  yang  2020/6/22 14:05
// @Update  yang  2020/6/22 14:05
package main

// 疑问：
/*
	1, Serial() 的时候，进行 双 sha256 计算后，到底要不要进行 小端转换 ??
 */

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"
)

type Block struct {
	Height int32   // 区块高度
	Hash []byte    // 区块头hash
	Header *BlockHeader  // 区块头
	Transactions []*Transaction  // 交易
}

type BlockHeader struct {
	Version int32    // 版本
	PrevHash []byte  // 引用的上一个区块 hash
	MerkleRoot []byte  // 默克尔根
	Time int32        // 时间戳
	Bits int32        // 目标难度值
	Nonce uint32      // 随机数
}

func (b *Block) String() string {
	var infos  []string
	infos = append(infos, fmt.Sprintf("----------------BLOCK----------------\n"))
	infos = append(infos, fmt.Sprintf("height : %d\n", b.Height))
	infos = append(infos, fmt.Sprintf("hash : %x\n", b.Hash))
	infos = append(infos, fmt.Sprintf("version : %d\n", b.Header.Version))
	infos = append(infos, fmt.Sprintf("PrevHash : %x\n", b.Header.PrevHash))
	infos = append(infos, fmt.Sprintf("MerkleRoot : %x\n", b.Header.MerkleRoot))
	infos = append(infos, fmt.Sprintf("time : %d\n", b.Header.Time))
	infos = append(infos, fmt.Sprintf("bits : %d\n", b.Header.Bits))
	infos = append(infos, fmt.Sprintf("nonce : %d\n", b.Header.Nonce))
	for tIndex,tx := range b.Transactions {
		infos = append(infos,fmt.Sprintf("Transaction[%d], TXID : %s\n", tIndex, hex.EncodeToString(tx.ID)))
		for tinIndex,tin := range tx.Vin {
			infos = append(infos,fmt.Sprintf("-- TXInput[%d]: \n", tinIndex))
			infos = append(infos,fmt.Sprintf("-- TxID : %s\n", hex.EncodeToString(tin.TXid)))
			infos = append(infos,fmt.Sprintf("-- Tin VoutIndex : %d\n", tin.VoutIndex))
			infos = append(infos,fmt.Sprintf("-- Tin Signature : %x\n", tin.Signature))
			infos = append(infos,fmt.Sprintf("-- Tin Pubkey : %x\n", tin.Pubkey))
		}

		for toutIndex,vout := range tx.Vout {
			infos = append(infos, fmt.Sprintf("\n"))
			infos = append(infos,fmt.Sprintf("-- TXOutput[%d]: \n", toutIndex))
			infos = append(infos,fmt.Sprintf("-- Tout value : %d\n", vout.Value))
			infos = append(infos,fmt.Sprintf("-- Tout PubkeyHash : %x\n", vout.PubkeyHash))
		}
	}
	return strings.Join(infos,"")
}
// Block 序列化
func (b *Block) Serialize() []byte {
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)

	err := enc.Encode(b)
	if err != nil {
		log.Panicf("Block Seriallize failed : ",err)
	}

	return encoded.Bytes()
}
// Block 反序列化
func DeserializeBlock(d []byte) *Block {
	var block Block
	decode := gob.NewDecoder(bytes.NewReader(d))
	err := decode.Decode(&block)
	if err != nil {
		log.Panicf("Block Deserialize failed : ",err)
	}

	return &block
}
// 根据区块中的所有交易，计算 默克尔根
func calcMerkleRoot(transactions []*Transaction) []byte{

	var datas [][]byte
	for _, tx := range  transactions {
		datas = append(datas, tx.ID)
	}

	// 默克尔根 hash : 根据每一笔交易 的 txid hash 构建默克尔树，计算出 默克尔根 hash
	merkleRoot := NewMerkleTree(datas)

	// 大小端转换
	ReverseBytes(merkleRoot.RootNode.Data)
	merkleRootHash := merkleRoot.RootNode.Data
	return merkleRootHash
}

// 根据上一区块 hash 创建新的 区块
func NewBlock(transactions []*Transaction, prevHash []byte, height int32) *Block {

	// 通过 transactions  计算出 默克尔根 hash
	merkleRoot := calcMerkleRoot(transactions)

	// 区块头
	header := BlockHeader{
		Version:    2,
		PrevHash:   prevHash,
		MerkleRoot: merkleRoot,
		Time:       int32(time.Now().Unix()),
		Bits:       486604799,
		Nonce:      0,
	}

	// 完整区块
	block := Block{
		Height:			height,   // 区块高度
		Hash:         []byte{},   // 区块头hash
		Header:       &header,    // 区块头
		Transactions: transactions, // 交易
	}

	pow := NewPOW(&block)
	nonce, data := pow.Run2()

	pow.block.Header.Nonce = nonce
	pow.block.Hash = data

	if pow.Validate() {
		fmt.Println("验证成功，创建一个新的区块成功！")
		return &block
	}

	return nil
}

// 创建一个创世区块
func NewGensisBlock(minerBTCAddress string) *Block {

	// 创建一个 coinbase 交易
	coinbaseTX := NewCoinbaseTX(minerBTCAddress,genesisData)

	datas := [][]byte {
		coinbaseTX.ID,
	}

	// 默克尔根 hash : 根据每一笔交易 的 txid hash 构建默克尔树，计算出 默克尔根 hash
	merkleRoot := NewMerkleTree(datas)

	// 大小端转换
	ReverseBytes(merkleRoot.RootNode.Data)
	merkleRootHash := merkleRoot.RootNode.Data

	header := BlockHeader{
		Version:    2,
		PrevHash:   []byte{},
		MerkleRoot: merkleRootHash,
		Time:       int32(time.Now().Unix()),
		Bits:       486604799,
		Nonce:      0,
	}

	block := Block{
		Height:			0,         // 区块高度 (创世区块高度 为 0)
		Hash:         []byte{},   // 区块头hash
		Header:       &header,    // 区块头
		Transactions: []*Transaction{coinbaseTX}, // 交易
	}

	// NewPOW 内部实现，简易了目标难度，用于测试。。。
	pow := NewPOW(&block)
	nonce, data := pow.Run2()

	pow.block.Header.Nonce = nonce
	pow.block.Hash = data

	if pow.Validate() {
		fmt.Println("验证成功，创世区块：")
		return &block
	}

	fmt.Println("验证失败!")
	return nil
}