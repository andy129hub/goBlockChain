// @Title  
// @Description  
// @Author  yang  2020/6/23 16:12
// @Update  yang  2020/6/23 16:12
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

type CLI struct {
	bc *BlockChain
}

// 打印命令使用说明
func (cli *CLI) printUsage() {
	fmt.Println("Usage(使用说明)")
	fmt.Println("-- minerlock : 添加区块")
	fmt.Println("-- blockinfo : 打印区块信息")
	fmt.Println("-- getbalance --address  : 输出账户余额")
	fmt.Println("-- send --from  --to    : 转账")
	fmt.Println("-- createwallet : 创建钱包")
	fmt.Println("-- listaddress : 显示所有钱包账户")
	fmt.Println("-- lastheight : 显示最新区块的高度")
	fmt.Println("-- startserver --nodeID --miner : 启动节点服务")
	os.Exit(1)
}

// 验证命令行参数是否有效
func (cli *CLI) validateArgs() {
	if len(os.Args) < 1 {
		os.Exit(1)
	}
	fmt.Println(os.Args)
}

// 执行程序解析命令行
func (cli *CLI) Run() {
	cli.validateArgs()

	/*   可使用 go 语言内置 os 包，设置环境变量来 存储，读取某些值
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Println("NODE_ID is not set.")
		os.Exit(1)
	}
	*/

	startServerCMD := flag.NewFlagSet("startserver", flag.ExitOnError)
	nodeParam := startServerCMD.String("nodeID","","server node ID")
	minerParam := startServerCMD.String("miner","","local node miner address")


	minerBlockCMD := flag.NewFlagSet("minerblock", flag.ExitOnError)
	blockInfoCMD := flag.NewFlagSet("blockinfo", flag.ExitOnError)
	getBalanceCMD := flag.NewFlagSet("getbalance", flag.ExitOnError)
	getBalanceParam := getBalanceCMD.String("address", "", "the address to get balance of")

	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	sendFromParam := sendCmd.String("from","","Source wallet address")
	sendToParam := sendCmd.String("to","", "Destination wallet address")
	sendAmoutParam := sendCmd.Int("amount",0, "Amount to send")

	createWalletCMD := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressCMD := flag.NewFlagSet("listaddress", flag.ExitOnError)

	getLastHeightCMD := flag.NewFlagSet("lastheight", flag.ExitOnError)


	if os.Args[1] != "startserver" {
		if cli.bc == nil {
			blockChain := InitBlockChain()
			cli.bc = blockChain
			// 代表数据库中没有 区块信息
			if cli.bc.tip == nil {
				wallets, err := NewWallets()
				var minerBTCAddress string
				if err != nil && os.IsNotExist(err) {
					minerBTCAddress = wallets.CreateWallet()
					gensisBlock := NewGensisBlock(minerBTCAddress)
					err := cli.bc.AddBlock(gensisBlock)
					if err != nil {
						log.Panic("Add gensis Block failed , err : ", err)
					}
					// 更新UTXO SET
					set := &UTXOSet{cli.bc}
					set.Reindex()
				}
			}
		}
	}

	// os.Args[0] 为 程序的名称 (或者 脚本的名称)
	switch os.Args[1] {
	case "minerblock":
		err := minerBlockCMD.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("命令解析失败")
			cli.printUsage()
		}

	case "blockinfo":
		err := blockInfoCMD.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("命令解析失败")
			cli.printUsage()
		}
	case "getbalance":
		err := getBalanceCMD.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("命令解析失败")
			cli.printUsage()
		}
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("命令解析失败")
			cli.printUsage()
		}
	case "createwallet":
		err := createWalletCMD.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("命令解析失败")
			cli.printUsage()
		}
	case "listaddress":
		err := listAddressCMD.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("命令解析失败")
			cli.printUsage()
		}
	case "lastheight":
		err := getLastHeightCMD.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("命令解析失败")
			cli.printUsage()
		}
	case "startserver":
		err := startServerCMD.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("命令解析失败")
			cli.printUsage()
		}
	default :
		cli.printUsage()
	}

	// 解析成功
	if minerBlockCMD.Parsed() {
		cli.bc.MinerBlock([]*Transaction{})
	}
	if blockInfoCMD.Parsed() {
		cli.bc.printBlockChain()
		return
	}
	if getBalanceCMD.Parsed() {

		if *getBalanceParam == "" {
			cli.printUsage()
		}
		fmt.Println("address : ", *getBalanceParam)

		start := time.Now().UnixNano()
		balance := cli.bc.getBalance(*getBalanceParam)
		end := time.Now().UnixNano()
		fmt.Printf("   balance from block: %d, time : %d\n",balance, end-start)   // balance from block: 50, time : 1000100

		start2 := time.Now().UnixNano()
		balance2 := cli.bc.getBalanceFromUTXOSet(*getBalanceParam)
		end2 := time.Now().UnixNano()
		fmt.Printf("   balance from db: %d, time : %d\n",balance2, end2-start2)   // balance from db: 50, time : 0

		/*
			通过 getBalance() 与 getBalanceFromUTXOSet() 的对比可知， 遍历整个区块找出UTXO 并计算余额，与 从数据库中找出UTXO并计算余额， 差别是很大的。

			通过数据库提前把 区块中所有的 UTXO 插入到数据库中，然后从 数据库中查询 pubkeyHash 对应的 UTXO 并计算余额，高效。
		 */

	}
	if sendCmd.Parsed() {
		if *sendFromParam == "" || *sendToParam == "" || *sendAmoutParam <=0 {
			cli.printUsage()
		}

		cli.send(*sendFromParam,*sendToParam,*sendAmoutParam)
	}
	if createWalletCMD.Parsed() {
		cli.createWallet()
	}
	if listAddressCMD.Parsed() {
		cli.listAddress()
	}
	if getLastHeightCMD.Parsed() {
		lastHeight := cli.bc.getLashHeight()
		fmt.Printf("lastHeight : %d\n", lastHeight)
	}
	if startServerCMD.Parsed() {
		if *nodeParam == "" {
			cli.printUsage()
		}

		cli.startServer(*nodeParam, *minerParam)

	}
}

func (cli *CLI) startServer(nodeID string, miner string) {
	fmt.Printf("Start server , nodeID : %s\n", nodeID)

	if len(miner) > 0 {
		if ValidateAddress([]byte(miner)) {
			fmt.Printf("miner address : %s\n", miner)
		}else {
			fmt.Println("miner address is invalid!")
			// cli.printUsage()
		}
	}

	StartServer(nodeID)
}

func (cli *CLI) send(from,to string, amount int) {

	tx,err := NewUTXOTransaction(from,to,amount,cli.bc)
	if err != nil {
		return
		fmt.Println("SEND FAILED!")
	}

	newBlock,err:= cli.bc.MinerBlock([]*Transaction{tx})
	if err != nil {
		log.Panic(err)
		return
	}

	set := UTXOSet{bchain:cli.bc}
	set.update(newBlock)
	fmt.Println("SEND SUCCESS!")

	a := cli.bc.getBalanceFromUTXOSet(from)
	fmt.Printf("address : %s,  balance : %d\n", from, a)

	b := cli.bc.getBalanceFromUTXOSet(to)
	fmt.Printf("address : %s,  balance : %d\n", to, b)

}

func (cli *CLI) createWallet() {
	// 初始化 钱包存储器，加载所有钱包账户
	wallets,err := NewWallets()
	if !os.IsNotExist(err) && err != nil {
		log.Panic("初始化 钱包存储器失败")
	}
	// 新创建一个钱包地址，并将地址添加到 钱包存储器中
	address := wallets.CreateWallet()   // 创建一个钱包地址之后，默认就将钱包存储在本地文件中 (包含 ： wallets.SaveToFile())
	// 将新的钱包地址通过 钱包存储器存放到文件中
	// wallets.SaveToFile()

	fmt.Printf("address : %s\n", address)
}

func (cli *CLI) listAddress() {
	// 初始化 钱包存储器，加载所有钱包账户
	wallets,err := NewWallets()
	if os.IsNotExist(err) {
		fmt.Println("没有有效地址！")
		cli.printUsage()
	}
	if err != nil {
		log.Panic("初始化 钱包存储器失败")
	}
	addresses := wallets.GetAddresses()
	if len(addresses) == 0 {
		fmt.Println("没有有效地址！")
		cli.printUsage()
		return
	}
	for _, address := range addresses {
		fmt.Println("address : ", address)
	}
}






