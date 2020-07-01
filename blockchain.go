// @Title  
// @Description  
// @Author  yang  2020/6/23 15:01
// @Update  yang  2020/6/23 15:01
package main

// boltdb --->  K/V 型数据库
/*
	Please note that Bolt obtains a file lock on the data file so multiple processes cannot open the same database at the same time.
	Opening an already open Bolt database will cause it to hang until the other process closes it.
		To prevent an indefinite wait you can pass a timeout option to the Open() function:

		db, err := bolt.Open("my.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
*/
import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"log"
	"os"
)

const dbFile = "blockchain.db"
const blockBucket = "blocks"

const minerAddress = "141DANcpwCBY7UvCdNVxSmiLxNPneAzueR"
// 12rqJLjfowrumPBMP4wNc9NKsYbg4wiSww
// 1DMyMVJJ5hNCpKkNnkYieUJcGEkkijKxTT
// 18rNG8sdeyLDoa88896vMnbXkEkLCsVnYj

const genesisData = "andy blockchain"

// BlockChain 代表区块链结构体
type BlockChain struct {
	tip []byte // 最近的一个区块的 hash 值
	db *bolt.DB
}
// BlockChain 迭代器
type BlockChainIterator struct {
	currentHash []byte
	db *bolt.DB
}
// 判断当前DB 文件是否存在
func IsDbExist() bool{
	_, err := os.Stat(dbFile)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// 挖矿
func (bc *BlockChain) MinerBlock(transactions []*Transaction) (*Block,error){

	// 验证交易
	for txIndex,tx := range transactions {
		if bc.VerifyTransaction(tx) == false {
			fmt.Printf("Transaction[%d]验证失败！\n", txIndex)
			// log.Panic("Transaction验证失败！")
			return nil,errors.New("Transaction验证失败！")
		}
	}

	fmt.Println("交易验证成功，开始打包交易至新的区块！")

	var prevHash []byte
	var lastHeight int32
	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockBucket))
		prevHash = b.Get([]byte("l"))
		blockData := b.Get(prevHash)
		prevBlock := DeserializeBlock(blockData)
		lastHeight = prevBlock.Height
		return nil
	})
	if err != nil {
		// log.Panic(err)
		return nil, err
	}

	newBlock := NewBlock(transactions, prevHash,lastHeight+1)
	if newBlock == nil {
		// log.Panic("创建新的区块失败")
		return nil, err
	}

	err = bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockBucket))
		err := b.Put(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			return err
		}

		err = b.Put([]byte("l"), newBlock.Hash)
		if err != nil {
			return err
			//log.Panic(err)
		}

		bc.tip = newBlock.Hash
		return nil
	})

	if err != nil {
		// panic(err)
		return nil, err
	}
	return newBlock, nil
}

// 初始化区块链
func InitBlockChain() *BlockChain {
	var tip []byte
	db,err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panicf("bolt open db failed : %v", err)
	}

	err = db.Update(func(tx *bolt.Tx) error{
		b := tx.Bucket([]byte(blockBucket))
		// 区块链不存在，则创建
		if b == nil {
			fmt.Println("区块链不存在，创建一个新的区块链")
			_, err := tx.CreateBucket([]byte(blockBucket))
			if err != nil {
				return err
				//log.Panic(err)
			}
		}else {   // 区块链存在，则取值
			tip = b.Get([]byte("l"))  // 最近一个区块的 hash
			//fmt.Printf("get, tip : %x\n", tip)
		}
		return nil
	})

	if err != nil {
		log.Panicf("err : %v", err)
	}

	//fmt.Printf("initBlockChain start, tip : %x\n", tip)
	bc := BlockChain{
		tip: tip,
		db:  db,
	}

	// 更新 数据库中的 ALL UTXO 数据
	if tip != nil {
		set := &UTXOSet{&bc}
		set.Reindex()
	}

	//fmt.Printf("initBlockChain end, tip : %x\n", tip)
	//fmt.Printf("initBlockChain end, bc.tip : %x\n", bc.tip)
	return &bc
}


// 测试使用
func InitBlockChainByTest(minerBTCAddress string) *BlockChain {
	var tip []byte
	db,err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panicf("bolt open db failed : %v", err)
	}

	err = db.Update(func(tx *bolt.Tx) error{
		b := tx.Bucket([]byte(blockBucket))
		// 区块链不存在，则创建
		if b == nil {
			fmt.Println("区块链不存在，创建一个新的区块链")
			// 构建一个创世区块
			genesis := NewGensisBlock(minerBTCAddress)
			b, err := tx.CreateBucket([]byte(blockBucket))
			if err != nil {
				return err
				//log.Panic(err)
			}
			err = b.Put(genesis.Hash, genesis.Serialize())
			if err != nil {
				return err
				//log.Panic(err)
			}
			err = b.Put([]byte("l"), genesis.Hash)
			if err != nil {
				return err
				//log.Panic(err)
			}
			tip = genesis.Hash
			//fmt.Printf("init, tip : %x\n", tip)
		}else {   // 区块链存在，则取值
			tip = b.Get([]byte("l"))  // 最近一个区块的 hash
			//fmt.Printf("get, tip : %x\n", tip)
		}
		return nil
	})

	if err != nil {
		log.Panicf("err : %v", err)
	}

	//fmt.Printf("initBlockChain start, tip : %x\n", tip)
	bc := BlockChain{
		tip: tip,
		db:  db,
	}

	// 更新 数据库中的 ALL UTXO 数据
	set := &UTXOSet{&bc}
	set.Reindex()

	//fmt.Printf("initBlockChain end, tip : %x\n", tip)
	//fmt.Printf("initBlockChain end, bc.tip : %x\n", bc.tip)
	return &bc
}

func (bc *BlockChain) iterator() *BlockChainIterator{
	//fmt.Printf("iterator, tip : %x\n", bc.tip)
	return &BlockChainIterator{
		currentHash: bc.tip,
		db: bc.db,
	}
}
func (bci *BlockChainIterator) next() *Block {
	//fmt.Printf("---------enter--------\n")
	var block *Block
	err := bci.db.View(func(tx *bolt.Tx) error {
		b:= tx.Bucket([]byte(blockBucket))
		//fmt.Printf("bci.currentHash : %x\n", bci.currentHash)
		deBlock := b.Get(bci.currentHash)   // 通过 hash 值找到 区块二进制编码
		//fmt.Printf("deBlock data : %x\n", deBlock)
		block = DeserializeBlock(deBlock)   // 通过二进制编码 反序列化 block 对象
		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	bci.currentHash = block.Header.PrevHash    // 将 bci 的 currentHash 设置为 当前区块的 上一个区块的 hash， 从而实现 一个区块一个区块的遍历


	//  测试失败，失败，失败，依旧 bci.currentHash 为 000000000000000,  原因是  iterator() 里面 bc.tip 就为 0000000000,
	// 但 测试可以得知，是 InitBlockChain()    set := &UTXOSet{&bc}  set.Reindex()   这里操作之后，导致了 bc.tip 为 00000000


	// 当 bci.currentHash 为零时，代表迭代到了创世区块，所以我们把 bci.currentHash 重置为 最新区块hash, 方便其他地方调用该函数。
	// 否则，当其他地方调用的时候，deBlock := b.Get(bci.currentHash)  , deBlock 就会返回空
	if len(bci.currentHash) == 0 {
		//fmt.Printf("---------enter2--------\n")
		err := bci.db.View(func(tx *bolt.Tx) error {
			b:= tx.Bucket([]byte(blockBucket))
			bci.currentHash = b.Get([]byte("l"))   // 通过 hash 值找到 区块二进制编码
			return nil
		})

		if err != nil {
			log.Panic(err)
		}

		//fmt.Printf("22222 bci.currentHash : %x\n", bci.currentHash)
	}

	return block
}

/*
	 通过命令行执行命令： goBlockChain.exe blockinfo   的时候发现，  bci.next() 调用了两次。最后调试得知，
			blockChain := InitBlockChain()
			cli := CLI{bc:blockChain}
			cli.Run()

	InitBlockChain() 里面结尾 set := UTXOSet{&bc}  set.Reindex()， 也会迭代区块链
	cli.Run()  -->  printBlockChain() 又迭代了一次，所以导致了两次，这就出现了多个地方使用  BlockChainIterator 的情况

		所以针对 BlockChainIterator 中的 currentHash 状态变量的值 被修改的问题，我们进行了如下修复：
		1， 迭代完成后，将 currentHash 重置为 最新区块 hash 值，具体建 next() 方法中的修改。

		2， 目前暂时修复，但是我感觉这里还是有一定的 风险性， 多处地方修改同一个值的问题。

				??????????  后续优化 ???????
 */
func (bc *BlockChain) printBlockChain() {
	bci := bc.iterator()

	for {
		fmt.Printf("-----------1--------------\n")
		block := bci.next()
		fmt.Printf("-----------2--------------\n")
		fmt.Println(block)

		fmt.Printf("len prevhash : %d\n", len(block.Header.PrevHash))
		// 到达创世区块
		if len(block.Header.PrevHash) == 0 {
			break
		}
	}
}

// 根据 公钥hash 寻找该地址对应的 未被花费的交易
/*	获取 公钥hash 对应的余额 === 原理如下：
	一：获取 公钥hash 对应的 UTXO （未被花费的交易）,  核心重点如下：
		1, 能被 公钥hash 验证
		2, 未被花费的交易
			a. 未被其他交易引用的 交易输出 ，也就是该 交易输出 没有作为 其他交易的 交易输入。

 */
func (bc *BlockChain) FindUnspentTrans(pubkeyhash []byte) []Transaction{
	var unspentTxs []Transaction

	spendTxOuts := make(map[string][]int)

	bci := bc.iterator()

	for {
		block := bci.next()

		// 从最近的区块往 创世区块 遍历
		for _,tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)
		output:
			for outIndex,out := range tx.Vout {
				if spendTxOuts[txID] != nil {
					for _, spentOut := range spendTxOuts[txID] {
						// 判断该交易输出 是否作为 其他交易的 交易输入 (就是被花费了)
						if spentOut == outIndex {
							continue output
						}
					}
				}
				// 程序运行到这里  代表 该交易输出，没有作为其他交易的 交易输入，有两种情况:
				/*
					1, if spendTxOuts[txID] != nil 不满足这个条件， 则代表 该交易中的任何一个输出都没被其他 交易作为 交易输入引用， 例如： 最后一个区块打包的交易数据
					2, 满足 if spendTxOuts[txID] != nil 这个条件，但不满足 spentOut == outIndex ，也就是该交易中的 交易输出，有些被引用了，有些没有被引用，没有被引用的 则就是 未花费的交易输出
				 */
				//然后再判断是否为能被 address 验证
				if out.CanBeUnlockedWith(pubkeyhash) {
					unspentTxs = append(unspentTxs, *tx)
				}
			}
			// 迭代交易输入  (非 coinbase 交易，因为  coinbase 交易没有 交易输入)
			// 交易输入都代表是 被引用的 交易输出
			if tx.IsCoinBase() == false {
				for _,in := range tx.Vin {
					// 判断是否为 address 验证
					if in.CanUnlockOutputWith(pubkeyhash) {
						inTxID := hex.EncodeToString(in.TXid)
						// 将被引用的交易ID 对应 引用该交易的 第几笔输出
						spendTxOuts[inTxID] = append(spendTxOuts[inTxID], in.VoutIndex)  // 花费的交易ID，绑定对应的 vout index
					}
				}
			}
		}
		if len(block.Header.PrevHash) == 0 {
			break
		}
	}

	// fmt.Println(unspentTxs)

	return unspentTxs
}

// (遍历整个区块)查询公钥hash 对应的UTXO (UTXO: 未被花费的 交易输出)
func (bc *BlockChain) FindUTXO(pubkeyhash []byte) []TXOutput{

	var UTXOs []TXOutput

	unspentTrans := bc.FindUnspentTrans(pubkeyhash)
	for _,tx := range unspentTrans {
		for _,out := range tx.Vout {
			if out.CanBeUnlockedWith(pubkeyhash) {
				UTXOs = append(UTXOs, out)
			}
		}
	}
	return UTXOs
}
// 获取 address 对应的余额数 (查询数据库获得)
func (bc *BlockChain) getBalanceFromUTXOSet(address string) int{

	decoderAddress := Base58Decode([]byte(address))
	pubkeyHash := decoderAddress[1:len(decoderAddress)-4]
	balance :=0

	utxoSet := UTXOSet{bc}
	UTXOs := utxoSet.FindUTXObyPubkeyHash(pubkeyHash)

	for _,out := range UTXOs {
		balance += out.Value
	}
	return balance
}


// 获取 address 对应的余额数 (遍历区块获得)
func (bc *BlockChain) getBalance(address string) int{

	decoderAddress := Base58Decode([]byte(address))
	pubkeyHash := decoderAddress[1:len(decoderAddress)-4]
	balance :=0
	UTXOs := bc.FindUTXO(pubkeyHash)
	for _,out := range UTXOs {
		balance += out.Value
	}
	return balance
}

// 查找 公钥hash 对应的  足够支付 amount 金额的  的所有未花费的交易输出  (Spendable： 可花费的)
func (bc *BlockChain) FindSpendableOutputs(pubkeyhash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)   // 交易ID --> 该交易中所有 未被花费的交易输出 index   (表示该交易中，所有可以被 pubkeyhash 验证的 总共多少笔 交易输出可以被使用)
	unspentTXs := bc.FindUnspentTrans(pubkeyhash)  // 找出包含 address 的未被花费的交易

	accumulated := 0

	// 从交易中 找出可以被 address 验证的 交易输出  (交易输出中的金额，会被拿来进行转账消费)
	Work:
	for _,tx := range unspentTXs {
		txID := hex.EncodeToString(tx.ID)
		for outIndex,out := range tx.Vout {
			// accumulated 数值大于 转账的金额的时候，则不再记录 可以被花费的交易输出 （代表找到了足够支付转账金额的 交易输出）
			if out.CanBeUnlockedWith(pubkeyhash) && accumulated < amount {
				accumulated += out.Value
				unspentOutputs[txID] = append(unspentOutputs[txID], outIndex)

				if accumulated >= amount {
					break Work   // 一旦 金额满足要求，则直接所有跳出查找工作，避免无意义的遍历工作。
					// break label   跳出循环，不再执行 for 循环代码
				}
			}
		}
	}

	return accumulated, unspentOutputs
}
// 对交易进行签名  (找到本次交易的 交易输入，通过交易输入中的 txID 找到区块链中被引用的 交易输出)
func (bc *BlockChain) SignTransaction(tx *Transaction, privateKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)
	for _,vin := range tx.Vin {
		prevTX,err :=  bc.FindTransactionByID(vin.TXid)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}
	tx.Sign(prevTXs, privateKey)
}

// 通过 Transaction 中交易输入中的 txID, 找到区块链中被引用的 交易输出 Transaction
func (bc *BlockChain) FindTransactionByID(ID []byte) (Transaction,error){
	bci := bc.iterator()

	for {
		block := bci.next()
		for _,tx := range block.Transactions {

			if bytes.Compare(tx.ID,ID) == 0 {
				return *tx,nil
			}
		}
		if len(block.Header.PrevHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("Can't find TX by ID!")

}

func (bc *BlockChain) VerifyTransaction(tx *Transaction) bool {
	prexTXs := make(map[string]Transaction)
	for _, vin := range tx.Vin {
		prevTx,err := bc.FindTransactionByID(vin.TXid)
		// 该交易的交易输入 引用的交易ID 在区块链中找不到，则代表此交易非法
		if err != nil {
			fmt.Println("not find prexTX from ID")
			return false
		}
		prexTXs[hex.EncodeToString(prevTx.ID)] = prevTx
	}
	return tx.Verfiy(prexTXs)
}

func (bc *BlockChain) FindAllUTXO() map[string]TXOutputs{
	utxoMap := make(map[string]TXOutputs)
	spentTXs := make(map[string][]int)

	bci := bc.iterator()

	for {

		block := bci.next()

		for _,tx := range block.Transactions {

			txID := hex.EncodeToString(tx.ID)
			Outputs:
			for outIndex,out := range tx.Vout {
				if spentTXs[txID] != nil {
					for _, spendOutIDs := range spentTXs[txID] {
						if spendOutIDs == outIndex {
							continue Outputs
						}
					}
				}
				// 程序运行到这里  代表 该交易输出，没有作为其他交易的 交易输入，有两种情况:
				/*
					1, if spendTXs[txID] != nil 不满足这个条件， 则代表 该交易中的任何一个输出都没被其他 交易作为 交易输入引用， 例如： 最后一个区块打包的交易数据
					2, 满足 if spendTXs[txID] != nil 这个条件，但不满足 spendOutIDs == outIndex ，也就是该交易中的 交易输出，有些被引用了，有些没有被引用，没有被引用的 则就是 未花费的交易输出
				*/
				outs := utxoMap[txID]
				outs.Outputs = append(outs.Outputs, out)
				utxoMap[txID] = outs
			}

			if tx.IsCoinBase() == false {
				for _,in := range tx.Vin {
					inTXID := hex.EncodeToString(in.TXid)
					spentTXs[inTXID] = append(spentTXs[inTXID], in.VoutIndex)
				}
			}
		}

		if len(block.Header.PrevHash) == 0 {
			break
		}
	}
	return utxoMap
}

// 获取最新区块的高度
func (bc *BlockChain) getLashHeight() int32{
	// bc.tip 为nil ,则代表本地数据库没有数据 (根据 tip 来判断 不是 100% 的确保数据库没数据，但这里才用一下)
	if bc.tip == nil {
		return -1
	}
	var lastBlock Block
	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockBucket))
		lastHash := b.Get([]byte("l"))

		blockData := b.Get(lastHash)
		lastBlock = *DeserializeBlock(blockData)
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	return lastBlock.Height
}

// 获取所有区块 hash 值
func (bc *BlockChain) GetBlockHashs() [][]byte {
	var blocks [][]byte
	bci := bc.iterator()

	for {
		block := bci.next()
		blocks = append(blocks, block.Hash)

		if len(block.Header.PrevHash) == 0 {
			break
		}
	}
	return blocks
}

// 通过 区块hash 值查找 对应区块信息
func (bc *BlockChain) GetBlockByHashID(hash []byte) (Block,error) {
	var block Block

	err := bc.db.View(func (tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockBucket))
		blockData := b.Get(hash)
		if blockData == nil {
			return errors.New("blockdata is nil ")
		}

		block = *DeserializeBlock(blockData)
		return nil
	})
	if err != nil {
		return block, err
	}

	return block,nil
}
// 从其他节点 同步区块信息，添加进数据库
func (bc *BlockChain) AddBlock(block *Block) error{
	err := bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockBucket))
		blockData := b.Get(block.Hash)
		// 如果区块已经存在，则不需要进行其他操作
		if blockData != nil {
			return nil
		}

		blockData = block.Serialize()
		err := b.Put(block.Hash, blockData)
		if err != nil {
			return err
		}

		// 获取当前数据库中 存储的最高区块信息
		lastHash := b.Get([]byte("l"))
		// 当 lastHash 不为空时，则代表本地数据库中有区块信息，则比较两个区块的 高度
		if lastHash != nil {
			lastBlockHash := b.Get(lastHash)
			lastBlock := DeserializeBlock(lastBlockHash)
			// 更新数据库中最高区块 数据 (根据区块高度大小)
			if block.Height > lastBlock.Height {
				err := b.Put([]byte("l"),block.Hash)
				if err != nil {
					return err
				}
				bc.tip = block.Hash    // 更新 bc.tip  (当前最高区块hash )
			}
		}else {  // 能走到这里的，代表是添加 第一个区块 (有可能是创世区块，也有可能是其他区块)， 在 端口3000 启动服务时，添加的是 创世区块， 在其他节点同步区块数据时，添加的是 最高区块
			b.Put([]byte("l"),block.Hash)
			bc.tip = block.Hash
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}