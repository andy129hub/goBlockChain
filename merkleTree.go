// @Title  
// @Description  
// @Author  yang  2020/6/22 14:27
// @Update  yang  2020/6/22 14:27
package main

import (
	"crypto/sha256"
	"fmt"
)

// 遇到的问题：
// 我们不能在 NewMerkleNode() 函数中进行 大小端操作
/*
	按照 merkleRootDemo 中的示例， 从 blockchain explore 网页中tx1, tx2 必须要先进行大小端的转换，
	再进行 两次hash, 最后得到的结果再进行反转才能得到正确的 merkle根 hash 值， 那为什么在这里就不行了呢？  得不到正确的结果。

	那是因为， 如果 交易笔数为 奇数，那么最后一个交易会和自身进行 拼接，然后进行 两次 hash 运算，问题就出现在这里！！！
		当是奇数的时候， left 和 right 就是一样的叶子节点，  那么他们的 data 数据都是一样的，我们看看下面的操作

			ReverseBytes(left.Data)
			ReverseBytes(right.Data)

		将一个数组进行 反转了之后，下面又反转回去了 （操作的是同一个 data ），所以 这样的 hash 值是不正确的。
*/


type MerkleTree struct {
	RootNode *MerkleNode
}


type MerkleNode struct {
	Left *MerkleNode
	Right *MerkleNode
	Data []byte
}


func NewMerkleNode(left,right *MerkleNode, data []byte) *MerkleNode{
	mNode := MerkleNode{}
	// 构建叶子节点
	if left == nil && right == nil {
		mNode.Data = data
	}else {   // 构建父节点

		/*
			按照 merkleRootDemo 中的示例， 从 blockchain explore 网页中tx1, tx2 必须要先进行大小端的转换，
			再进行 两次hash, 最后得到的结果再进行反转才能得到正确的 merkle根 hash 值， 那为什么在这里就不行了呢？  得不到正确的结果。

			那是因为， 如果 交易笔数为 奇数，那么最后一个交易会和自身进行 拼接，然后进行 两次 hash 运算，问题就出现在这里！！！
				当是奇数的时候， left 和 right 就是一样的叶子节点，  那么他们的 data 数据都是一样的，我们看看下面的操作

					ReverseBytes(left.Data)
					ReverseBytes(right.Data)

				将一个数组进行 反转了之后，下面又反转回去了 （操作的是同一个 data ），所以 这样的 hash 值是不正确的。
		*/

		prevHashes := append(left.Data, right.Data...)
		firsthash := sha256.Sum256(prevHashes)
		secondhash := sha256.Sum256(firsthash[:])
		mNode.Data = secondhash[:]

		fmt.Println("---------------------------------------------------")
		fmt.Printf("left : %x\n", left.Data)
		fmt.Printf("right : %x\n", right.Data)

		fmt.Printf("calc : %x\n", mNode.Data)
		fmt.Println("---------------------------------------------------")

	}

	mNode.Left = left
	mNode.Right = right
	return &mNode
}

// 构建默克尔树 （由下往上构建）
func NewMerkleTree(datas [][]byte) *MerkleTree {
	var nodes []MerkleNode
	for _,data := range datas {
		node := NewMerkleNode(nil,nil,data)
		nodes = append(nodes,*node)
	}

	j:=0
	// 控制树的 深度
	for nSize := len(datas);nSize >1;nSize = (nSize+1)/2 {
		// i+=2 ， 跨越式 两两hash
		for i:=0;i<nSize;i+=2 {
			// 交易笔数的 奇，偶情况
			i2 := Min(i+1, nSize-1)
			// 相邻两个作为 left,right， 并进行hash 运算得到 一个父节点
			node := NewMerkleNode(&nodes[j+i], &nodes[j+i2], nil)   // 还有这里是 j+i,  不是 j+1
			// 将 两个叶子节点 hash 运算得到的父节点 追加到 nodes 集合中
			// 最后加入nodes 里面的 一定是 根节点也就是 默克尔根
			nodes = append(nodes, *node)

			// fmt.Printf("node ： %x\n", node.Data)
		}
		// j 可以理解为 一层, 最底层叶子节点 相邻两个节点 hash 完成之后， 上到上一层，对父节点进行两两 hash
		j += nSize
	}

	merkleRoot := MerkleTree{
		&(nodes[len(nodes)-1]),
	}

	return &merkleRoot
}
