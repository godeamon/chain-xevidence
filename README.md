# chain-xevidence
xuperchain block hash evidence

编译：make

编译产出：output/xevidence

配置文件：output/conf/config.yaml

启动：cd output 后，./xevidence



### config

```yaml
# endorseService Info
# testNet addrs
# endorseServiceHost: "39.156.69.83:37100"
endorseServiceHost: "127.0.0.1:37101"
complianceCheck:
  # 是否需要进行合规性背书
  isNeedComplianceCheck: false
  # 是否需要支付合规性背书费用
  isNeedComplianceCheckFee: false
  # 合规性背书费用
  complianceCheckEndorseServiceFee: 400
  # 支付合规性背书费用的收款地址
  complianceCheckEndorseServiceFeeAddr: aB2hpHnTBDxko3UoP2BpBZRujwhdcAFoT
  # 如果通过合规性检查，签发认证签名的地址
  complianceCheckEndorseServiceAddr: jknGxa6eyum1JrATWvSJKW3thJ9GKHA9n
#创建平行链所需要的最低费用
minNewChainAmount: "100"
crypto: "xchain"
txVersion: 1
# maxRecvMsgSize set the max message size in bytes the server can receive.
# If this is not set, gRPC uses the default 4MB.
maxRecvMsgSize: 134217728

######### 以下配置为，将 sideChain 的区块哈希存储到 mainChain 的 XEvidence 系统合约中
mainChain:
  name: xuperos # 业务上的链名字
  url: 127.0.0.1:37101
  heightInterval: 1
  # SL1jzovziZ1vgEVifbUGBSFamzYrdQiXp
  senderToAccount: #key为地址，和 sideChain.nodeToSender 的 key 对应
    SL1jzovziZ1vgEVifbUGBSFamzYrdQiXp:
      accountMnemonic: 留 趋 护 露 孙 雏 损 委 罩 然 筑 给
      accountMnemonicLanguage: 1
    SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co:
      accountMnemonic: 留 趋 护 露 孙 雏 损 委 罩 然 筑 给
      accountMnemonicLanguage: 1
    default: # 必须设置 default，当矿工不在配置文件内时使用 default 发送存证交易
      accountMnemonic: 留 趋 护 露 孙 雏 损 委 罩 然 筑 给
      accountMnemonicLanguage: 1
sideChain:
  name: xasset # 业务上的链名字
  url: 127.0.0.1:37101
  startHeigh: 0 #起始高度
  safeHeightInterval: 3 #不会回滚高度
  xChainVerison: 5 # 2：xchain2，5：xchain5.x版本，不支持其他配置，且目前只支持5
  chainName: xuper # XuperChain的平行链名字
  nodeToSender: # key 为矿工地址，value为发送交易地址，即不同矿工的区块使用不同账户发送存证交易
    TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY: SL1jzovziZ1vgEVifbUGBSFamzYrdQiXp

```

