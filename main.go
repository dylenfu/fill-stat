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
)

const (
	LRC_SYMBOL = "LRC"
	LRC_ADDR = "0xEF68e7C694F40c8202821eDF525dE3782458639f"
	)

var (
	rds *dao.RdsService
	event extractor.EventData
	globalConfig *config.GlobalConfig
	oldTokens = make(map[common.Hash]string)
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

	// load old tokens
	oldTokens[common.HexToHash("0x2956356cD2a2bf3202F771F50D3D14A367b48070")] = "WETH_OLD"
	oldTokens[common.HexToHash("0x86Fa049857E0209aa7D9e616F7eb3b3B78ECfdb0")] = "EOS"
	oldTokens[common.HexToHash("0xf5B3b365FA319342e89a3Da71ba393E12D9F63c3")] = "FOO"
	oldTokens[common.HexToHash("0xb5f64747127be058Ee7239b363269FC8cF3F4A87")] = "BAR"

	// load market util
	util.Initialize(&globalConfig.Market)

	// load loopring contract accessor
	err := accessor.Initialize(globalConfig.Accessor)
	err = loopringaccessor.Initialize(globalConfig.LoopringProtocol)
	if nil != err {
		log.Fatalf("what fucking idiot err:%s", err.Error())
	}
	for name, e := range loopringaccessor.ProtocolImplAbi().Events {
		if name != contract.EVENT_RING_MINED  {
			continue
		}
		event.Id = e.Id()
		event.Name = e.Name
		event.Abi = loopringaccessor.ProtocolImplAbi()
		event.Event = &contract.RingMinedEvent{}
		log.Infof("extractor,contract event name:%s -> key:%s", event.Name, event.Id.Hex())
	}

	stat(&globalConfig.Item)
}

func stat(item *config.ItemOption) {
	start := rds.FindLatestId(item.DbName)
	if start == 0 {
		start = item.Start
	} else {
		start += 1
	}
	if item.End <= start {
		log.Fatalf("fuck scanning end error")
	}

	log.Debugf("start:%d", start)
	for i := item.Start; i<=item.End; i++ {
		if ring, err := rds.FindRingMinedById(i); err != nil {
			log.Errorf(err.Error())
		} else {
			fills := getFills(ring.TxHash)
			for _, v := range fills {
				single(v, item.DbName)
			}
		}
	}
}

func getFills(txhash string) []*dao.FillEvent {
	var (
		recipient ethtyp.TransactionReceipt
		tx ethtyp.Transaction
		list []*dao.FillEvent
	)

	retry := 10

	for i:=0;i<retry;i++ {
		if err := accessor.GetTransactionReceipt(&recipient, txhash, "latest"); err != nil {
			log.Errorf("retry to get transaction recipient, retry count:%d", i+1)
		}
		if err := accessor.GetTransactionByHash(&tx, txhash, "latest"); err != nil {
			log.Errorf("retry to get transaction, retry count:%d", i+1)
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
		if common.HexToHash(v.Topics[0]) == event.Id {
			evtLog = v
		}
	}
	if len(evtLog.Data) < 100 {
		return list
	}

	data := hexutil.MustDecode(evtLog.Data)
	for _, topic := range evtLog.Topics {
		decodeBytes := hexutil.MustDecode(topic)
		decodedValues = append(decodedValues, decodeBytes)
	}
	event.Abi.UnpackEvent(event.Event, event.Name, data, decodedValues)

	src := event.Event.(*contract.RingMinedEvent)

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

func single(fill *dao.FillEvent, dbName string) error {
	lrcFee, _ := new(big.Int).SetString(fill.LrcFee, 0)
	lrcReward, _ := new(big.Int).SetString(fill.LrcReward, 0)
	lrcTotal := new(big.Int).Add(lrcFee, lrcReward)

	// tokenS
	tokenS := common.HexToAddress(fill.TokenS)
	symbolS, _ := util.GetSymbolWithAddress(tokenS)
	amountS, _ := new(big.Int).SetString(fill.AmountS, 0)
	splitS, _ := new(big.Int).SetString(fill.SplitS, 0)
	amountS = new(big.Int).Add(amountS, splitS)
	if strings.ToUpper(symbolS) == LRC_SYMBOL {
		amountS = new(big.Int).Add(amountS, lrcTotal)
	}
	setStat(fill.TokenS, symbolS, dbName, amountS.String(), fill.TxHash, fill.ID)

	// tokenB
	tokenB := common.HexToAddress(fill.TokenB)
	symbolB, _ := util.GetSymbolWithAddress(tokenB)
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

	return nil
}

func setStat(token, symbol, dbName, addAmount, latestTxHash string, latestFillId int) {
	data, err := rds.FindStatDataByToken(token)
	if err != nil {
		data.Amount = addAmount
		data.LatestId = latestFillId
		data.LatestDb = dbName
		data.Token = token
		data.Symbol = symbol
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
	}
	sum, _ := new(big.Int).SetString(amount, 0)
	readableAmount := new(big.Rat).SetFrac(sum, decimal).FloatString(2)
	return readableAmount
}

func setAllReadbleAmount() {
	list, _ := rds.GetAllStatData()
	for _, v := range list {
		if v.Symbol == "" {
			if symbol, ok := oldTokens[common.HexToHash(v.Token)]; ok {
				v.Symbol = symbol
			}
		}
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