// @Title  
// @Description  
// @Author  yang  2020/6/22 17:18
// @Update  yang  2020/6/22 17:18
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strconv"
)

var (
	MAX_NONCE uint32 = math.MaxUint32
	MAX_INTERNAL uint32 = 10000000
	resultChan chan int32
)

type POW struct {
	block *Block
	target *big.Int
}

func NewPOW(b *Block) *POW{

	/*   正式版挖矿，计算难度值
	bits := b.Header.Bits
	str := strconv.FormatInt(int64(bits), 16)
	//fmt.Println(str)   // 1d00ffff
	data,_:= hex.DecodeString(str)
	target := calcTarget2(data)

	t,_ := hex.DecodeString(string(target))    // 将 target 字符串形式的 16进制 转换为  []byte 的 16进制

	var targetHash = big.NewInt(0)
	targetHash.SetBytes(t)   // 为了和 serial() 返回的 序列化 data 比较，targetHash.setBytes() 必须传入 16进制的 []byte
	//fmt.Printf("%x\n", targetHash)
	resultChan = make(chan int32)
	 */

	// 测试....
	targetInt := big.NewInt(1)
	targetInt.Lsh(targetInt, uint(256-16))   // 左移多少位  （降低难度值，测试使用....）
	//fmt.Printf("targetInt : %x\n", targetInt)

	pow := POW{
		b,
		targetInt,
	}
	return &pow
}
// 拼接区块信息 (版本，前一区块hash, 默克尔根，时间戳，目标难度值，随机数) ，方便后续 与 目标难度 hash 进行比较
func (pow *POW) prepareData(nonce uint32) []byte {
	result := bytes.Join([][]byte{IntToHex(pow.block.Header.Version),
		pow.block.Header.PrevHash,
		pow.block.Header.MerkleRoot,
		IntToHex(pow.block.Header.Time),
		IntToHex(pow.block.Header.Bits),
		UIntToHex(nonce)},
		[]byte{}) // 拼接各个 []byte

	firstHash := sha256.Sum256(result)
	blockHeaderHash := sha256.Sum256(firstHash[:])

	ReverseBytes(blockHeaderHash[:])
	return blockHeaderHash[:]
}


// 开始计算随机数 (测试：。。。)
func (pow *POW) Run2() (uint32, []byte){

	var nonce uint32
	var headerHash = big.NewInt(0)

	var data []byte
	for nonce < MAX_NONCE {
		data = pow.prepareData(nonce)
		headerHash.SetBytes(data)
		//fmt.Printf("target : %064x\n",pow.target)
		//fmt.Printf("nonce %d , hash : %064x\n",nonce, headerHash)
		if headerHash.Cmp(pow.target) == -1 {

			//fmt.Printf("找到随机数：%d\n", nonce)
			//fmt.Printf("区块难度hash：%064x\n", pow.target)
			//fmt.Printf("区块头 hash ：%064x\n", headerHash)
			pow.block.Hash = data
			pow.block.Header.Nonce = nonce
			break
		}
		nonce++
	}

	return nonce, data
}
// 验证区块是否有效
func (pow *POW) Validate() bool {
	var hashInt big.Int
	data := pow.prepareData(pow.block.Header.Nonce)
	hashInt.SetBytes(data)
	isValid := hashInt.Cmp(pow.target) == -1

	return isValid
}

// 尝试使用 goroutine 多线程进行挖矿
func (pow *POW) startWork() {

	start := MAX_NONCE-1
	end := start - MAX_INTERNAL
	var i int32 = 0
	for i=0;i<=int32(((MAX_NONCE-1)/MAX_INTERNAL));i++ {
		go pow.Run(i, start, end)

		start =  end - 1
		if start < MAX_INTERNAL {
			end = 0
		}else {
			end = start - MAX_INTERNAL
		}
	}

	select {
		case result := <-resultChan: {
			fmt.Printf("协程 %d : 挖到！", result)
			break
		}
	}
}

func (pow *POW) Run(index int32, start,end uint32) {

	var nonce uint32
	nonce = start
	var headerHash = big.NewInt(0)
	if nonce >=0 && nonce < MAX_NONCE {
		for nonce >= end {
			data := pow.prepareData(nonce)
			headerHash.SetBytes(data)
			fmt.Printf("协程[%d],nonce[%d--%d]: %d , hash : %x\n", index, start,end,nonce, data)
			if headerHash.Cmp(pow.target) == -1 {

				fmt.Printf("协程[%d] 找到随机数：%d\n",index, nonce)
				fmt.Printf("区块难度hash：%064x\n", pow.target)
				fmt.Printf("区块头 hash ：%064x\n", headerHash)
				pow.block.Hash = data
				pow.block.Header.Nonce = nonce

				resultChan <- index
				break
			}
			nonce--
		}

		fmt.Printf("协程[%d], nonce[%d--%d] 未找到！\n", index, start,end)

	}
}

// 使用 公式求 target
// 目标位 = 系数* 2 ^ (8 * (指数-3))
// result = coefficient * 2 ^(8 * (exponent-3))
func calcTarget2(bits []byte) []byte{

	// 第一个字节代表 指数
	exponent := bits[:1]
	//fmt.Printf("%x\n", exponent)
	// 后面的字节代表系数
	coefficient := bits[1:]
	//fmt.Printf("%x\n",coefficient)

	// 将 exponent 转换为 字符串形式的 16进制
	str := hex.EncodeToString(exponent)
	// base:16 代表 str 为16进制, bitSize:8, 代表转换为 int8 类型
	exp, _ := strconv.ParseInt(str, 16, 8)
	//fmt.Println("exp : ", exp)

	str2 := hex.EncodeToString(coefficient)
	coe,_ := strconv.ParseInt(str2, 16, 32)

	//fmt.Println("coe : ", coe)

	num1 := 8 * (exp-3)
	//fmt.Println("num1 : ",num1)

	var a = big.NewInt(2)
	var b = big.NewInt(num1)
	power := Powerf(a,b)  // 计算 a 的 b 次方 （big.Int 类型）

	var coeInt = big.NewInt(coe)
	power.Mul(power,coeInt)

	//fmt.Printf("%x\n", power.Bytes())  // 1b7b74000000000000000000000000000000000000000000
	//fmt.Printf("%d\n", len(power.Bytes()))  // 24

	// 目前还剩下的问题是， 要将 24位 前面补零 至 32位(还要补 16个 0)，有没有什么简单的办法。 (不使用 bytes.repeat)
	// 解决方案：
	/*
	    通过 fmt.Sprintf("%064x", power)   格式化拼接字符串, 将 power 的10进制值 以 %x(16进制)格式化输出，
	  	输出 64个长度 (没2个长度为 一个字节，共32个字节)，并且前面必须补 0
	*/

	target := fmt.Sprintf("%064x", power)
	// fmt.Printf("格式化： %s\n", target)

	// targetHash,_ := hex.DecodeString(target)
	return []byte(target)


	// big.Int .Text(base int) 可以将10进制的结果 以字符串的形式显示出来
	/*
		fmt.Printf("%d\n", power)      // 673862533877092685902494685124943911912916060357898797056
		fmt.Println("power : ",power)  // 673862533877092685902494685124943911912916060357898797056
		fmt.Println(power.Text(10))   // 673862533877092685902494685124943911912916060357898797056
		fmt.Println(power.Text(16))   //  1b7b74000000000000000000000000000000000000000000
	*/

}
