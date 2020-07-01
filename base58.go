// @Title  
// @Description  
// @Author  yang  2020/6/25 11:42
// @Update  yang  2020/6/25 11:42
package main

import (
	"bytes"
	"math/big"
)

var base58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")


// Base58Encode base58 加密
func Base58Encode(input []byte) []byte {

	// 用于存储编码后的结果
	var result []byte

	// 把字节数组 input 转化为一个 大整数
	x := big.NewInt(0).SetBytes(input)
	// 长度 58 的大整数
	base := big.NewInt(int64(len(base58Alphabet)))
	// 0 的大整数
	zero := big.NewInt(0)
	// 大整数的 指针  (用于接收 取模后的 余数)
	mod := &big.Int{}

	/*	math/big/ini.go

		// Cmp compares x and y and returns:
		//
		//   -1 if x <  y
		//    0 if x == y
		//   +1 if x >  y
		//
	*/
	// x 与  zero 进行比较  (x, zero 都是 *Int 类型)
	// x
	for x.Cmp(zero) !=0 {
		x.DivMod(x,base,mod)  // x 对 base(也就是 58) 取余, x 保存 整数，mod 保存余数， 直到x 被除尽为 0
		result = append(result, base58Alphabet[mod.Int64()])   // 根据余数到 base58 编码中找到对应的 编码字符
	}

	// 举例: (二进制转换， base58 其实是 4 % 58 的操作，下面以 二进制举例)
	/*	10进制 4 , 转换为 二进制

		4  % 2  == 2  ...    0      append 0  -->  result 0
		2  % 2  == 1  ...    0      append 0  -->  result 00
		1  % 2  == 0  ...    1      append 1  -->  result 001

		so  result ==  001
		但是 4 的二进制应该是  100,  所以我们需要进行一次反转。
	*/

	// 反转 []byte
	ReverseBytes(result)

	// 假如 []byte 里面 前面几个 都为 0 , 则替换为 1  (比特币中特殊的写法)
	for _, b := range input {
		// fmt.Println("b : ",b)
		// 前面 0x00 代表 比特币主网 版本 , input 是[]byte 类型，里面存放的是  256位数据，
		// b 为一个字节(8位) 在 256位长度的数据中 单个长度代表4位，两个长度就代表8位一个字节，所以 可以和 0x00 比较。
		if b == 0x00 {   // 则将其追加到 result 前面，用 base58Alphabet[0] 也就是 '1' 代替
			// 如果 []byte
			result = append([]byte{base58Alphabet[0]}, result...)
		}else {
			break
		}
	}

	return result
}

// Base58Decode base58 解码
func Base58Decode(input []byte) []byte{
	result := big.NewInt(0)
	zeroBytes := 0
	for _, b := range input {
		// 只判断前方有多少个 1
		if b == base58Alphabet[0] {
			zeroBytes++
		}else{
			break
		}
	}

	payload := input[zeroBytes:]
	for _,b := range payload {
		charIndex := bytes.IndexByte(base58Alphabet, b)   // 反推出 余数
		result.Mul(result, big.NewInt(58))    // 之前的结果乘以  58，  结果用 result 保存
		result.Add(result, big.NewInt(int64(charIndex)))   // 在加上这个余数

		/*	示例，以二进制示例


		 */

		// fmt.Printf("result : %v\n", result.Int64())
	}

	// 将 result 转换为 字节数组
	decoded := result.Bytes()

	// fmt.Println("decoded1 : ", string(decoded))

	decoded = append(bytes.Repeat([]byte{0x00}, zeroBytes), decoded...)

	// fmt.Println("decoded2 : ", string(decoded))

	return decoded
}
