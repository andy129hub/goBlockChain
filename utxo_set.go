// @Title  
// @Description  
// @Author  yang  2020/6/27 18:43
// @Update  yang  2020/6/27 18:43
package main

import (
	"encoding/hex"
	"github.com/boltdb/bolt"
	"log"
)

const utxoBucket = "chainset"
type UTXOSet struct {
	bchain *BlockChain
}

// 重置 数据库中的 ALL UTXO 数据
func (u *UTXOSet) Reindex() {
	db := u.bchain.db
	bucketName := []byte(utxoBucket)

	err := db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket(bucketName)
		// 如果删除的时候出现错误，并且这个错误不是 bucket 未找到的错误，那么则代表出现了其他 不可预测的错误，则 panic
		if err != nil && err != bolt.ErrBucketNotFound {
			log.Panic(err)
		}

		_,err = tx.CreateBucket(bucketName)
		if err != nil {
			log.Panic(err)
		}

		return nil
	})

	if err != nil {
		log.Panic(err)
	}
	// 遍历区块，查找 所有的 UTXO， 并在下面将其 写入数据库
	allUTXO := u.bchain.FindAllUTXO()

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		for txID, outs := range allUTXO {
			key,err := hex.DecodeString(txID)
			if err != nil {
				log.Panic(err)
			}

			err = b.Put(key, outs.Serialize())
			if err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}
// 从数据库中查找 能被 pubkeyHash 验证的 UTXO （可以通过这个方法，查询地址余额）
func (u UTXOSet) FindUTXObyPubkeyHash(pubkeyHash []byte) []TXOutput {
	var UTXOs []TXOutput
	db := u.bchain.db

	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(utxoBucket))
		c := bucket.Cursor()

		for k,v := c.First();k!=nil;k,v = c.Next() {
			outs := DeserializeTXOutputs(v)
			for _,out := range outs.Outputs {
				if out.CanBeUnlockedWith(pubkeyHash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return UTXOs
}
// 当新创建一个区块后，更新 UTXO SET (不需要全部遍历一遍，然后更新，而是使用如下方法，快速更新，省时)
/*	核心两步:
1, 更新该笔交易中交易输入引用的交易ID 对应的 UTXO (可能都被引用，可能只引用几笔，所以需要用一个 updateouts 集合存储未被花费的 交易输出，然后更新到 数据库中)
2，将新区块中交易的 交易输出存储到数据库
 */
func (u UTXOSet) update(block *Block) {
	db := u.bchain.db
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		// 遍历该区块中的所有交易
		for _, tx := range block.Transactions {
			if tx.IsCoinBase() == false {
				// 遍历该笔交易中所有的 交易输入
				for _,vin := range tx.Vin {
					updateouts := TXOutputs{}
					// 通过交易输入引用的交易ID，在数据库中查询到 被引用的交易
					outsByte := b.Get(vin.TXid)
					// 序列化该笔交易的 交易输出
					outs := DeserializeTXOutputs(outsByte)
					// 遍历 引用的交易的交易输出
					for outIndex, out := range outs.Outputs {
						// 如果交易输入 引用了某个交易输出，则代表，引用的交易中的交易输出被 花费，不计入统计，
						// 当不等于的时候，则代表 交易输出未被引用，则如下，添加到一个临时集合中
						if outIndex != vin.VoutIndex {
							// 将 未花费的交易输出，整合在一起
							updateouts.Outputs = append(updateouts.Outputs, out)
						}
					}
					// 如果 updateouts.Outputs 临时集合里面的元素为0， 则代表没有 未被花费的交易 (该交易输入引用了该笔交易的所有输出)
					if len(updateouts.Outputs) == 0 {
						err := b.Delete(vin.TXid)   // 删除该交易输入引用的 交易ID 对应的UTXO
						if err != nil {
							log.Panic(err)
						}
					}else {
						// 更新 交易输入引用的交易ID 对应的 UTXO
						err := b.Put(vin.TXid, updateouts.Serialize())
						if err != nil {
							log.Panic(err)
						}
					}
				}
			}

			// 当程序运行到这里，代表对引用的交易ID 对应的 UTXO 都更新完毕
			// 接下来是对新添加区块中交易的 UTXO 进行数据库存储
			// 新区块中的 所有交易输出，必定都是 UTXO （未被花费的输出）
			newOutputs := TXOutputs{}
			for _,out := range tx.Vout {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}

			err := b.Put(tx.ID, newOutputs.Serialize())
			if err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}



