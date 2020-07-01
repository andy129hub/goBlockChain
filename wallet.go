// @Title  
// @Description  
// @Author  yang  2020/6/25 11:44
// @Update  yang  2020/6/25 11:44
package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"golang.org/x/crypto/ripemd160"
)

const version = byte(0x00)
type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey []byte
}

func NewWallet() *Wallet {
	private,public := newKeyPair()
	wallet := Wallet{
		PrivateKey: private,
		PublicKey:  public,
	}
	return &wallet
}

func (w *Wallet) GetAddress() []byte{
	// double hash
	publicKeyHash := doubleHashPubkey(w.PublicKey)
	// Base58Check
	// 1, 散列 (版本号前缀+原信息)
	versionPayload := append([]byte{version}, publicKeyHash...)
	// 2, 校验和
	checksum := checkSum(versionPayload)
	// 3 将校验和追加到 尾部
	fullPayload := append(versionPayload, checksum...)

	// Base58编码最终生成 比特币地址
	address := Base58Encode(fullPayload)
	return address
}

// 对publickey 进行 double hash 运算
func doubleHashPubkey(pubKey []byte) []byte{
	// 将公钥进行两次 hash 运算 (RIPEMD160(SHA256(pubkey)))
	// 1, SHA256
	pubKeyHash256 := sha256.Sum256(pubKey)
	// 2, PIPEMD160
	PIPEMD160Hasher := ripemd160.New()
	_,err := PIPEMD160Hasher.Write(pubKeyHash256[:])
	if err != nil {
		fmt.Println("PIPEMD160Hasher.Write failed : ", err)
	}
	publicRIPEMD160 := PIPEMD160Hasher.Sum(nil)
	return publicRIPEMD160
}

func checkSum(versionPayload []byte) []byte {
	// 2, 两次 SHA256 运算计算出 校验和并追加到 尾部
	// 2.1 两次 SHA256
	firstSHA := sha256.Sum256(versionPayload)
	secondSHA := sha256.Sum256(firstSHA[:])
	// 2.2 两次 SHA256 运算后，取头部前4位值 为校验和
	checksum := secondSHA[:4]
	return checksum
}
// 验证 address 是否有效
func ValidateAddress(address []byte) bool {
	pubkeyHash := Base58Decode(address)

	actualCheckSum := pubkeyHash[len(pubkeyHash)-4:]
	publickeyHash := pubkeyHash[1:len(pubkeyHash)-4]

	// 采取自己拼接 版本前缀 + publickeyHash 的操作，并进行验证 校验和
	targetChecksum := checkSum(append([]byte{version}, publickeyHash...))

	return bytes.Compare(targetChecksum, actualCheckSum) == 0

}

// ECDSA 椭圆曲线 ： y^2 = (x^3 + a * x + b) mod p
func newKeyPair() (ecdsa.PrivateKey, []byte) {
	// 生成椭圆曲线, secp256r1 曲线。     比特币当中的曲线是  secp256k1
	// elliptic : 椭圆
	// curve : 曲线
	curve := elliptic.P256()

	/*	crypto/ecdsa/ecdsa.go

		func GenerateKey(c elliptic.Curve, rand io.Reader) (*PrivateKey, error) {
	*/

	// private 为 PrivateKey 结构体
	/*
		// PrivateKey represents an ECDSA private key.
		type PrivateKey struct {
			PublicKey
			D *big.Int
		}
	*/
	// 其中 PublicKey 可以获取到 公钥， D 代表私钥
	private, err := ecdsa.GenerateKey(curve, rand.Reader)   // 通过 rand.Reader 随机数的种子，生成一个 椭圆曲线上的点

	if err != nil {
		fmt.Println("GenerateKey failed : ", err)
	}

	// 将 公钥 x 与 y 坐标 拼接起来 得到 公钥
	pubkey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	return *private, pubkey

}