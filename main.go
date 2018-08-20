package main

import (
	"github.com/dylenfu/fill-stat/config"
	"github.com/dylenfu/fill-stat/dao"
	util "github.com/Loopring/relay-lib/marketutil"
	"reflect"
	"github.com/Loopring/relay-lib/log"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strings"
	"github.com/Loopring/relay-lib/eth/accessor"
	"github.com/Loopring/relay-lib/eth/loopringaccessor"
	ethtyp "github.com/Loopring/relay-lib/eth/types"
	"github.com/Loopring/relay-lib/eth/contract"
	"github.com/Loopring/extractor/extractor"
	"github.com/Loopring/go-ethereum/common/hexutil"
	"github.com/Loopring/relay-lib/types"
	"github.com/Loopring/relay-lib/cache"
	"os"
)

const (
	LRC_SYMBOL = "LRC"
	LRC_ADDR = "0xEF68e7C694F40c8202821eDF525dE3782458639f"
	BLOCK_NUMBER_STR = "6137760"
	)

var (
	rds            *dao.RdsService
	ringMinedEvent extractor.EventData
	transferEvent extractor.EventData
	globalConfig *config.GlobalConfig
	oldTokens = make(map[common.Address]string)
)

func main() {
	// load config
	globalConfig = config.LoadConfig()
	if _, err := config.Validator(reflect.ValueOf(globalConfig).Elem()); nil != err {
		panic(err)
	}

	// init logger
	logger := log.Initialize(globalConfig.Log)
	defer func() {
		if nil != logger {
			logger.Sync()
		}
	}()

	// load mysql
	rds = dao.NewDb(&globalConfig.Mysql)
	cache.NewCache(globalConfig.Redis)

	// load old tokens
	oldTokens[common.HexToAddress("0x2956356cD2a2bf3202F771F50D3D14A367b48070")] = "WETH_OLD"
	oldTokens[common.HexToAddress("0x86Fa049857E0209aa7D9e616F7eb3b3B78ECfdb0")] = "EOS"
	oldTokens[common.HexToAddress("0xf5B3b365FA319342e89a3Da71ba393E12D9F63c3")] = "FOO"
	oldTokens[common.HexToAddress("0xb5f64747127be058Ee7239b363269FC8cF3F4A87")] = "BAR"
	oldTokens[common.HexToAddress("0xd2C6738D45b090ec05210fE8DCeEF4D8fc392892")] = "SET"

	// load market util
	util.Initialize(&globalConfig.Market)

	// load loopring contract accessor
	err := accessor.Initialize(globalConfig.Accessor)
	err = loopringaccessor.Initialize(globalConfig.LoopringProtocol)
	if nil != err {
		log.Fatalf("what fucking idiot err:%s", err.Error())
	}
	for name, e := range loopringaccessor.Erc20Abi().Events {
		if name != contract.EVENT_TRANSFER  {
			continue
		}

		switch name {
		//case contract.EVENT_RING_MINED:
		//	ringMinedEvent.Id = e.Id()
		//	ringMinedEvent.Name = e.Name
		//	ringMinedEvent.Abi = loopringaccessor.ProtocolImplAbi()
		//	ringMinedEvent.Event = &contract.RingMinedEvent{}

		case contract.EVENT_TRANSFER:
			transferEvent.Id = e.Id()
			transferEvent.Name = e.Name
			transferEvent.Abi = loopringaccessor.Erc20Abi()
			transferEvent.Event = &contract.TransferEvent{}
		}

		log.Infof("extractor,contract event name:%s -> key:%s", transferEvent.Name, transferEvent.Id.Hex())
	}

	stat(&globalConfig.Item)
}

func stat(item *config.ItemOption) {
	for i := item.Start; i<=item.End; i++ {
		log.Debugf("find ringmined event:%d", i)
		if ring, err := rds.FindRingMinedById(i); err != nil {
			log.Errorf(err.Error())
		} else {
			//fills := getFills(ring.TxHash)
			evts := getTransfer(ring.TxHash)
			processTransferList(evts, item.DbName)
			//for _, v := range evts {
			//	processSingleFill(v, item.DbName)
			//}
		}
	}
}

func oldStat(item *config.ItemOption) {
	for i := item.Start; i<=item.End; i++ {
		if fill, err := rds.FindFillById(i); err == nil {
			log.Debugf("mysql fill id:%d", i)
			processSingleFill(fill, item.DbName)
		}
	}
}

func getTransfer(txhash string) []*types.TransferEvent {
	var (
		recipient ethtyp.TransactionReceipt
		tx ethtyp.Transaction
		list []*types.TransferEvent
	)

	retry := 10

	for i:=0;i<retry;i++ {
		if err := accessor.GetTransactionReceipt(&recipient, txhash, BLOCK_NUMBER_STR); err != nil {
			log.Errorf("retry to get transaction recipient, retry count:%d", i+1)
		} else {
			break
		}
	}
	for i:=0; i<retry; i++ {
		if err := accessor.GetTransactionByHash(&tx, txhash, BLOCK_NUMBER_STR); err != nil {
			log.Errorf("retry to get transaction, retry count:%d", i+1)
		} else {
			break
		}
	}

	if len(recipient.Logs) < 1 {
		log.Debugf("cann't get ringmined event or tx is failed")
		return list
	}

	for _, v := range recipient.Logs {
		if common.HexToHash(v.Topics[0]).Hex() != transferEvent.Id.Hex() {
			log.Debugf("topic[0]:%s, transferId:%s", v.Topics[0], transferEvent.Id.Hex())
			continue
		}

		data := hexutil.MustDecode(v.Data)
		var decodedValues [][]byte
		for _, topic := range v.Topics {
			decodeBytes := hexutil.MustDecode(topic)
			decodedValues = append(decodedValues, decodeBytes)
		}
		transferEvent.Abi.UnpackEvent(transferEvent.Event, transferEvent.Name, data, decodedValues)

		src := transferEvent.Event.(*contract.TransferEvent)

		transfer := src.ConvertDown()
		txinfo := setTxInfo(&tx, recipient.GasUsed.BigInt(), big.NewInt(0), "submitRing")
		transfer.TxInfo = txinfo
		transfer.Protocol = common.HexToAddress(v.Address)
		list = append(list, transfer)
	}

	return list
}

func getFills(txhash string) []*dao.FillEvent {
	var (
		recipient ethtyp.TransactionReceipt
		tx ethtyp.Transaction
		list []*dao.FillEvent
	)

	retry := 10

	for i:=0;i<retry;i++ {
		if err := accessor.GetTransactionReceipt(&recipient, txhash, BLOCK_NUMBER_STR); err != nil {
			log.Errorf("retry to get transaction recipient, retry count:%d", i+1)
		} else {
			break
		}
	}
	for i:=0; i<retry; i++ {
		if err := accessor.GetTransactionByHash(&tx, txhash, BLOCK_NUMBER_STR); err != nil {
			log.Errorf("retry to get transaction, retry count:%d", i+1)
		} else {
			break
		}
	}

	if len(recipient.Logs) < 1 {
		log.Errorf("can not get ringmined event")
		return list
	}

	var (
		evtLog ethtyp.Log
		decodedValues [][]byte
	)
	for _, v := range recipient.Logs {
		if common.HexToHash(v.Topics[0]) == ringMinedEvent.Id {
			evtLog = v
		}
	}
	if len(evtLog.Data) < 100 {
		log.Errorf("ring mined event data length invalid")
		return list
	}

	data := hexutil.MustDecode(evtLog.Data)
	for _, topic := range evtLog.Topics {
		decodeBytes := hexutil.MustDecode(topic)
		decodedValues = append(decodedValues, decodeBytes)
	}
	ringMinedEvent.Abi.UnpackEvent(ringMinedEvent.Event, ringMinedEvent.Name, data, decodedValues)

	src := ringMinedEvent.Event.(*contract.RingMinedEvent)

	_, fills, err := src.ConvertDown()
	if err != nil {
		log.Errorf(err.Error())
		return list
	}

	txinfo := setTxInfo(&tx, recipient.GasUsed.BigInt(), big.NewInt(0), "submitRing")
	for _, v := range fills {
		v.TxInfo = txinfo
		var fill dao.FillEvent
		fill.ConvertDown(v)
		list = append(list, &fill)
	}

	return list
}

func processTransferList(list []*types.TransferEvent, dbName string) error {
	type stdata struct {
		symbol string
		token string
		txhash string
		amount *big.Int
	}
	stmp := make(map[string]*stdata)
	symbolCnt := 0
	for _, v := range list {
		symbol := getSymbol(v.Protocol.Hex())
		if st, ok := stmp[symbol]; !ok {
			st = &stdata{}
			st.symbol = symbol
			st.txhash = v.TxHash.Hex()
			st.token = v.Protocol.Hex()
			st.amount = v.Amount
			stmp[symbol] = st

			symbolCnt += 1
		} else {
			st.amount = new(big.Int).Add(st.amount, v.Amount)
		}
	}

	for _, v := range stmp {
		if v.symbol == LRC_SYMBOL && symbolCnt > 2 {
			log.Debugf("lrc gas or other")
		} else {
			setStat(v.token, v.symbol, dbName, v.amount.String(), v.txhash,1)
		}
	}
	return nil
}

func processSingleFill(fill *dao.FillEvent, dbName string) error {
	lrcFee, _ := new(big.Int).SetString(fill.LrcFee, 0)
	lrcReward, _ := new(big.Int).SetString(fill.LrcReward, 0)
	lrcTotal := new(big.Int).Add(lrcFee, lrcReward)

	// tokenS
	symbolS := getSymbol(fill.TokenS)
	amountS, _ := new(big.Int).SetString(fill.AmountS, 0)
	splitS, _ := new(big.Int).SetString(fill.SplitS, 0)
	amountS = new(big.Int).Add(amountS, splitS)
	if strings.ToUpper(symbolS) == LRC_SYMBOL {
		amountS = new(big.Int).Add(amountS, lrcTotal)
	}
	setStat(fill.TokenS, symbolS, dbName, amountS.String(), fill.TxHash, fill.ID)

	// tokenB
	symbolB := getSymbol(fill.TokenB)
	amountB, _ := new(big.Int).SetString(fill.AmountB, 0)
	splitB, _ := new(big.Int).SetString(fill.SplitB, 0)
	amountB = new(big.Int).Add(amountB, splitB)
	if strings.ToUpper(symbolB) == LRC_SYMBOL {
		amountB = new(big.Int).Add(amountB, lrcTotal)
	}
	setStat(fill.TokenB, symbolB, dbName, amountB.String(), fill.TxHash, fill.ID)

	// lrc
	if strings.ToUpper(symbolS) != LRC_SYMBOL && strings.ToUpper(symbolB) != LRC_SYMBOL {
		setStat(LRC_ADDR, LRC_SYMBOL, dbName, lrcTotal.String(), fill.TxHash, fill.ID)
	}

	log.Debugf("txhash:%s fill_%d orderhash:%s", fill.TxHash, fill.FillIndex, fill.OrderHash)
	return nil
}

func setStat(token, symbol, dbName, addAmount, latestTxHash string, latestFillId int) {
	data, err := rds.FindStatDataBySymbol(symbol)
	if err != nil {
		data.Amount = addAmount
		data.LatestId = latestFillId
		data.LatestDb = dbName
		data.Token = token
		data.Symbol = symbol
		data.TxCount = 1
		data.LatestTxHash = latestTxHash
		data.ReadableAmount = readableAmount(token, data.Amount)
		if err := rds.Add(&data); err != nil {
			log.Errorf(err.Error())
		} else {
			log.Debugf("insert symbol:%s, totalAmount:%s, fillId:%d", symbol,  data.Amount, latestFillId)
		}
	} else {
		current, _ := new(big.Int).SetString(data.Amount, 0)
		added, _ := new(big.Int).SetString(addAmount, 0)
		data.Amount = new(big.Int).Add(current, added).String()
		data.LatestDb = dbName
		data.LatestId = latestFillId
		if data.LatestTxHash != latestTxHash {
			data.TxCount += 1
		}
		data.LatestTxHash = latestTxHash
		data.ReadableAmount = readableAmount(token, data.Amount)
		if err := rds.Save(data); err != nil {
			log.Errorf(err.Error())
		} else {
			log.Debugf("update symbol:%s, totalAmount:%s, fillId:%d", symbol,  data.Amount, latestFillId)
		}
	}
}

func readableAmount(token,amount string) string {
	addr := common.HexToAddress(token)
	decimal := big.NewInt(1e18)
	if token, err := util.AddressToToken(addr); err == nil {
		decimal = token.Decimals
		if token.Symbol == "FUN" {
			log.Debugf("decimal ", decimal.String())
			os.Exit(1)
		}
	}
	sum, _ := new(big.Int).SetString(amount, 0)
	readableAmount := new(big.Rat).SetFrac(sum, decimal).FloatString(2)
	return readableAmount
}

func setAllReadbleAmount() {
	list, _ := rds.GetAllStatData()
	for _, v := range list {
		v.Symbol = getSymbol(v.Token)
		v.ReadableAmount = readableAmount(v.Token, v.Amount)
		rds.Save(v)
	}
}

func setTxInfo(tx *ethtyp.Transaction, gasUsed, blockTime *big.Int, methodName string) types.TxInfo {
	var txinfo types.TxInfo

	txinfo.Protocol = common.HexToAddress(tx.To)
	txinfo.From = common.HexToAddress(tx.From)
	txinfo.To = common.HexToAddress(tx.To)

	if impl, ok := loopringaccessor.ProtocolAddresses()[txinfo.To]; ok {
		txinfo.DelegateAddress = impl.DelegateAddress
	} else {
		txinfo.DelegateAddress = types.NilAddress
	}

	txinfo.BlockNumber = tx.BlockNumber.BigInt()
	txinfo.BlockTime = blockTime.Int64()
	txinfo.BlockHash = common.HexToHash(tx.BlockHash)
	txinfo.TxHash = common.HexToHash(tx.Hash)
	txinfo.TxIndex = tx.TransactionIndex.Int64()
	txinfo.Value = tx.Value.BigInt()

	txinfo.GasLimit = tx.Gas.BigInt()
	txinfo.GasUsed = gasUsed
	txinfo.GasPrice = tx.GasPrice.BigInt()
	txinfo.Nonce = tx.Nonce.BigInt()

	txinfo.Identify = methodName

	return txinfo
}

func getSymbol(token string) string {
	symbol, _ := util.GetSymbolWithAddress(common.HexToAddress(token))
	if len(symbol) > 1 {
		return symbol
	}
	if symbol, ok := oldTokens[common.HexToAddress(token)]; ok {
		return symbol
	} else {
		return ""
	}
}
