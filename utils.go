// @Title  
// @Description  
// @Author  yang  2020/6/22 14:08
// @Update  yang  2020/6/22 14:08
package main

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"time"
)


// 求 a,b 两值的最小值
func Min(a,b int) int {
	if a > b {
		return b
	}
	return a
}

// 反转  []byte (针对 tx hash, merkle root hash 进行大小端转换)
func ReverseBytes(data []byte) {
	for i,j := 0, len(data)-1; i<j; i,j = i+1,j-1 {
		data[i],data[j] = data[j],data[i]
	}
}

// 将 int 类型转换为 16进制小端模式
func IntToHex(num int32) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff,binary.LittleEndian,num)

	if err != nil {
		panic("IntToHex failed")
	}
	return buff.Bytes()
}

// 将 uint 类型转换为 16进制小端模式
func UIntToHex(num uint32) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff,binary.LittleEndian,num)

	if err != nil {
		panic("IntToHex failed")
	}
	return buff.Bytes()
}

func Int64ToHex(num uint64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff,binary.LittleEndian,num)

	if err != nil {
		panic("float64ToHex failed")
	}
	return buff.Bytes()
}


// 将日期字符串转换为 time.Time
func StringToDate(dateStr string) time.Time {
	date, err := time.Parse("2006-01-02 15:04:05", dateStr)
	if err != nil {
		panic("stringToDate failed")
	}
	// fmt.Println(date)
	return date
}


//  使用 big.Int 实现 x 的 n 次方 （对比如上）
func Powerf(x *big.Int, n *big.Int) *big.Int {
	var ans = big.NewInt(1)
	var zero = big.NewInt(0)
	var one = big.NewInt(1)
	var second = big.NewInt(2)

	var modRes = big.NewInt(0)

	for n.Cmp(zero) !=0 {
		if modRes.Mod(n,second).Cmp(one) == 0 {
			ans.Mul(ans,x)
		}
		x.Mul(x,x)
		n.Div(n,second)
	}
	return ans
}
