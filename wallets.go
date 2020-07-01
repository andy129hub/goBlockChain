// @Title  
// @Description  
// @Author  yang  2020/6/25 13:06
// @Update  yang  2020/6/25 13:06
package main

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"io/ioutil"
	"log"
	"os"
)

const walletFile = "wallet.dat"

type Wallets struct {
	WalletsStore map[string]*Wallet
}
// 创建钱包存储器 (存储多个钱包账户)
func NewWallets() (*Wallets, error) {
	wallets := Wallets{}
	wallets.WalletsStore = make(map[string]*Wallet)
	err := wallets.LoadFromFile()
	return &wallets, err
}

// 新建一个钱包地址
func (ws *Wallets) CreateWallet() string {
	wallet := NewWallet()
	address := string(wallet.GetAddress())
	ws.WalletsStore[address] = wallet
	ws.SaveToFile()   // 将钱包数据保存在本地文件中
	return address
}
// 获取所有钱包地址
func (ws *Wallets) GetAddresses() []string {
	var addresses []string
	for address,_ := range ws.WalletsStore {
		addresses = append(addresses, address)
	}

	return addresses
}

func (ws *Wallets) GetWallet(address string) *Wallet{
	return ws.WalletsStore[address]
}

// 将所有钱包地址序列化保存在 文件中
func (ws *Wallets) SaveToFile() {
	var content bytes.Buffer
	// 序列化之前，先注册一下 ws 中用到的接口类型
	gob.Register(elliptic.P256())
	encoder := gob.NewEncoder(&content)

	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}
	// 写入之前，会删除之前的内容，达到覆盖的作用
	err = ioutil.WriteFile(walletFile, content.Bytes(), 0777)
	if err != nil {
		log.Panic(err)
	}
}

// 从文件中加载所有钱包
func (ws *Wallets) LoadFromFile() error {
	// 判断文件是否存在,  注意 os.Stat()  和  os.IsNotExist() 的用法
	if _,err := os.Stat(walletFile);os.IsNotExist(err) {
		return err
	}
	fileContent,err := ioutil.ReadFile(walletFile)
	if err != nil {
		return err
	}

	var wallets Wallets
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		return err
	}

	ws.WalletsStore = wallets.WalletsStore

	return err
}