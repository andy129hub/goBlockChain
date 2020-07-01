// @Title  
// @Description  
// @Author  yang  2020/6/22 14:30
// @Update  yang  2020/6/22 14:30
package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
)

// 交易  (包含：交易ID，多个输入交易，多个输出交易)
type Transaction struct {
	ID []byte
	Vin []TXInput
	Vout []TXOutput
}

// 交易输入
type TXInput struct {
	TXid []byte     // 引用包含正在使用的 UTXO 交易的 ID
	VoutIndex int   // 标识 来自该交易中的第几个 UTXO 被引用
	Signature []byte
	Pubkey []byte   // 公钥
}
// 交易输出
type TXOutput struct {
	Value int
	PubkeyHash []byte // 公钥hash
}
// 交易输出集合
type TXOutputs struct {
	Outputs []TXOutput
}

const SubSidy = 50

func (tx Transaction) String() string {
	var lines []string
	lines = append(lines, fmt.Sprintf(" Transaction : %x", tx.ID))
	for i,input := range tx.Vin {
		lines = append(lines, fmt.Sprintf("Input : %d", i))
		lines = append(lines, fmt.Sprintf("  TXID : %x", input.TXid))
		lines = append(lines, fmt.Sprintf("  Out : %d", input.VoutIndex))
		lines = append(lines, fmt.Sprintf("  Signature : %x", input.Signature))
	}

	for i,output := range tx.Vout {
		lines = append(lines,fmt.Sprintf("Ouput : %d", i))
		lines = append(lines,fmt.Sprintf("  Value : %d", output.Value))
		lines = append(lines,fmt.Sprintf("  Script : %s",string(output.PubkeyHash)))
	}

	return strings.Join(lines, "\n")
}
// 序列化交易输出集合, 用于保存在 bolt 数据库中
func (outs TXOutputs) Serialize() []byte {
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(outs)
	if err != nil {
		fmt.Println("TXOutputs encoded err : ", err)
		log.Panic(err)
	}
	return encoded.Bytes()
}
// 反序列化交易输出
func DeserializeTXOutputs(data []byte) TXOutputs {
	var outputs TXOutputs
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&outputs)
	if err != nil {
		fmt.Println("TXOutputs decode err : ", err)
		log.Panic(err)
	}
	return outputs
}

func (out *TXOutput) Lock(address []byte) {
	decodeAddress := Base58Decode(address)
	pubkeyHash := decodeAddress[1:len(decodeAddress)-4]
	out.PubkeyHash = pubkeyHash
}

// 序列化交易
func (tx Transaction)Serialize() []byte {
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		fmt.Println("encode err : ", err)
		return nil
	}
	return encoded.Bytes()
}
// 计算 交易的 hash
func (tx *Transaction)Hash() []byte {
	txcopy := *tx
	txcopy.ID = []byte{}

	hash := sha256.Sum256(txcopy.Serialize())

	return hash[:]
}
// 根据 金额和地址 新建一个交易输出
func NewTXOutput(value int, address string) *TXOutput {
	txo := &TXOutput{value, nil}
	txo.Lock([]byte(address))
	return txo
}

// 新建一个 coinbase 交易  (to 代表矿工地址， data 代码创世区块 内置内容)
func NewCoinbaseTX(to ,data string) *Transaction{
	// coinbase 交易没有  输入交易，所以txin 的字段属性都设置为 空
	txin := TXInput{[]byte{}, -1, []byte(data), nil}
	txout := NewTXOutput(SubSidy,to)

	tx := Transaction{nil,[]TXInput{txin}, []TXOutput{*txout}}
	tx.ID = tx.Hash()
	return &tx
}

func (out *TXOutput) CanBeUnlockedWith(pubkeyhash []byte) bool {
	return bytes.Compare(out.PubkeyHash, pubkeyhash) == 0
}

func (in *TXInput) CanUnlockOutputWith(unlockdata []byte) bool {
	lockingHash := doubleHashPubkey(in.Pubkey)
	return bytes.Compare(lockingHash,unlockdata) == 0
}

// 判断是否为 coinbase 交易
func (tx Transaction) IsCoinBase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].TXid) == 0 && tx.Vin[0].VoutIndex == -1
}
// 转账
func NewUTXOTransaction(from,to string, amount int, bc *BlockChain) (*Transaction,error) {

	wallets, err := NewWallets()
	if err != nil {
		// log.Panic("转账失败！")
		return &Transaction{},err
	}
	// from 代表转账人的地址 (钱包客户端发起的转账交易)
	wallet := wallets.GetWallet(from)
	if wallet == nil {
		return &Transaction{},errors.New("wallet is nil")
	}

	pubkeyHash := doubleHashPubkey(wallet.PublicKey)

	var inputs []TXInput
	var outputs []TXOutput
	acc, validOutputs := bc.FindSpendableOutputs(pubkeyHash, amount)
	// validOutputs, 可能包含多笔 Transaction (引用多笔交易中的 交易输出 作为这次转账的 交易输入)
	if acc < amount {
		log.Panic("Error: Not Enough Funds!")
	}
	for txid,outs := range validOutputs {
		txID,err := hex.DecodeString(txid)
		if err != nil {
			// log.Panic(err)
			return &Transaction{},err
		}
		// 将每一笔可以被消费的 交易输出，作为即将进行转账的 交易输入
		for _,out := range outs {
			fmt.Printf("NewUTXOTransaction txinput :txID : %x\n", txID)
			input := TXInput{txID, out, nil, wallet.PublicKey}   // Pubkey 是未经过 hash 的 公钥
			inputs = append(inputs,input)
		}
	}
	// 交易输出(转账)，转账给 to
	outputs = append(outputs,*NewTXOutput(amount,to))

	// 交易输出(找零)，将余额(acc-amount) 转给 from (转账的人的地址)
	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc-amount,from))
	}

	// 构建一笔交易 (包含交易输入和交易输出)
	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	// 对这笔交易进行签名
	bc.SignTransaction(&tx, wallet.PrivateKey)
	return &tx,nil
}

// 对 Transaction 进行签名   (对副本进行特殊操作，计算出签名数据之后再赋值给 原数据，  这波操作。。。 有些难以理解。)
/*
	1, 先找到该 Transaction 中所有交易输入引用的 所有交易 (通俗点：就是 引用上一笔交易的 交易输出)。
	2，把引用的所有交易输出 进行副本拷贝 (具体操作见 TrimmedCopy(), 注意有特殊操作)
	3, (修改拷贝) 针对每一个交易输入 进行签名
		a, 将 该笔 Transaction 中的交易输入的 pubkey 属性 置为 交易输入引用的某笔交易输出的 pubkeyhash .
		b, 将整笔 Transaction 进行 序列化并 Hash()运算, 得到 transaction ID
		c, 根据 ID 进行  ecdsa.Sign 签名  (当然还需要 privateKey),  signature := append(r.Bytes(), s.Bytes()...)   // r,s 进行拼接，形成一次 完整的签名
	4, (修改原数据) 将计算出来的 签名, 赋值给 该笔交易输入的 Signature 字段。
	5，循环 3 的操作 （针对每一个交易输入都要计算得出签名，赋值给 原数据的 Signature 字段）。

 */
func (tx *Transaction) Sign(prevTXs map[string]Transaction, privateKey ecdsa.PrivateKey) {
	if tx.IsCoinBase() {
		return
	}
	// 验证交易是否有效 (原理：该Transaction 中的交易输入都 引用着区块链中的某笔 transaction 的交易输出)
	for _,vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.TXid)].ID == nil {
			log.Panic("error transaction")
		}
	}

	// 针对拷贝值进行操作
	txCopy := tx.TrimmedCopy()
	// 对该交易中的每一个 交易输入都进行签名
	for inIndex,vin := range txCopy.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.TXid)]
		txCopy.Vin[inIndex].Signature = nil
		txCopy.Vin[inIndex].Pubkey = prevTx.Vout[vin.VoutIndex].PubkeyHash  // 将 该笔 Transaction 中的交易输入的 pubkey 属性 置为 交易输入引用的某笔交易输出的 pubkeyhash .

		txCopy.ID = txCopy.Hash()   // 对该笔 Transaction 进行 hash
		r,s,err := ecdsa.Sign(rand.Reader, &privateKey, txCopy.ID)  // ecdsa.Sign() 通过私钥 对 txCopy.ID 进行签名
		if err != nil {
			log.Panic(err)
		}

		signature := append(r.Bytes(), s.Bytes()...)   // r,s 进行拼接，形成一次 完整的签名
		// 针对原始值进行操作 (签名之后，将签名赋值给 Transaction 中的交易输入的 Signature 属性)
		tx.Vin[inIndex].Signature = signature

		txCopy.Vin[inIndex].Pubkey = nil    // 重新置为 nil， 从而不影响全局
	}
}

// 交易的副本  （注意针对 交易输入，有特殊的操作，将 Signature, Pubkey 的值都置为 nil ）
func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	for _,vin := range tx.Vin {
		inputs = append(inputs, TXInput{vin.TXid, vin.VoutIndex, nil, nil })
	}

	for _,vout := range tx.Vout {
		outputs = append(outputs, TXOutput{vout.Value, vout.PubkeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs}
	return txCopy
}
// 验证交易  (重点理解，如何验证交易，涉及到的 算法************&&&&&&&&&&&&&&&&&&&&&)
func (tx *Transaction) Verfiy(prevTXs map[string]Transaction) bool {
	// 如果是 coinbase 交易则不需要验证，直接返回 true
	if tx.IsCoinBase() {
		fmt.Println("------------------------1111111-")
		return true
	}
	for _,vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.TXid)].ID  == nil {
			fmt.Println("------------------------222222-")
			return false
		}
	}

	// 交易副本
	txcopy := tx.TrimmedCopy()
	// 椭圆曲线
	curve := elliptic.P256()
	for inIndex,vin := range tx.Vin {   // 注意这里，下面还要使用 tx.Vin 中的 pubkey, 所以不能使用 txCopy.Vin (因为 txCopy中进行了TrimmedCopy() 特殊操作，将vin 的 pubkey 置为了nil)
		prevTx := prevTXs[hex.EncodeToString(vin.TXid)]
		fmt.Printf("txID : %s\n", hex.EncodeToString(vin.TXid))
		// 必须根据下标修改 TXInput, 如果修改 vin.Signature ，那么修改的只是 range 出来的 副本
		txcopy.Vin[inIndex].Signature = nil
		txcopy.Vin[inIndex].Pubkey = prevTx.Vout[vin.VoutIndex].PubkeyHash  // ???? 之所以这样做，是由于  Sign()中也是这样操作的，所以验证签名就要还原之前的操作
		txcopy.ID = txcopy.Hash()   // 这就是需要验证的内容

		r := big.Int{}
		s := big.Int{}

		siglen := len(vin.Signature)
		r.SetBytes(vin.Signature[:siglen/2])
		s.SetBytes(vin.Signature[siglen/2:])
		fmt.Printf("vin.Signature : %x\n", vin.Signature)

		x := big.Int{}
		y := big.Int{}

		keyLen := len(vin.Pubkey)   // 注意这个 Pubkey 是 tx.Vin 迭代出来的 vin ，而不是  txcopy.Vin
		x.SetBytes(vin.Pubkey[:keyLen/2])   // 也就是 PublicKey 的 x 坐标
		y.SetBytes(vin.Pubkey[keyLen/2:])   // 也就是 PublicKey 的 y 坐标
		fmt.Printf("vin.Pubkey : %x\n", vin.Pubkey)

		// 获取到公钥
		rawPubkey := ecdsa.PublicKey{curve, &x,&y}
		// 通过公钥 与 r,s 经过算法 验证 txcopy.ID 是否有效 （以上操作txcopy.Hash(), 就是还原 sign() 的操作）
		if ecdsa.Verify(&rawPubkey, txcopy.ID, &r,&s) == false {
			fmt.Println("-----------------------33333--")
			return false
		}

		txcopy.Vin[inIndex].Pubkey = nil  // 与Sign() 中操作一致
	}

	return true
}


