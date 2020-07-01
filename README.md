# goBlockChain

**项目说明：** Go语言开发区块链底层公链 

**使用：**<br>
**1, 编译代码：** go build . <br>
**2, 命令说明：**

   Usage(使用说明) <br>
  -- minerlock : 添加区块 <br>
  -- blockinfo : 打印区块信息 <br>
  -- getbalance --address  : 输出账户余额 <br>
  -- send --from  --to    : 转账 <br>
  -- createwallet : 创建钱包 <br>
  -- listaddress : 显示所有钱包账户 <br>
  -- lastheight : 显示最新区块的高度 <br>
  -- startserver --nodeID --miner : 启动节点服务 <br>

***详解： startserver 命令*** <br>
1, 编译代码之后，可直接执行 startserver --nodeID 3000  命令 <br>
   系统会自动 创建一个创世区块。也可在执行 startserver 命令之前，在本地进行 添加区块，转账等操作，创建多个区块。<br>
2, 在本地电脑的其他位置，再开启一个终端，模拟第二台服务节点. <br>
   执行 startserver --nodeID 3001 命令 <br>
3, 3001 端口服务节点会 自动去连接 3000端口的节点，进行 版本信息比对，区块信息获取等操作，自动同步 3000 端口的 区块信息。   
