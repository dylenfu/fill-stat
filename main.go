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
)

const (
	LRC_SYMBOL = "LRC"
	LRC_ADDR = "0xEF68e7C694F40c8202821eDF525dE3782458639f"
	)

func main() {
	globalConfig := config.LoadConfig()
	if _, err := config.Validator(reflect.ValueOf(globalConfig).Elem()); nil != err {
		panic(err)
	}
	logger := log.Initialize(globalConfig.Log)
	defer func() {
		if nil != logger {
			logger.Sync()
		}
	}()

	rds := dao.NewDb(&globalConfig.Mysql)
	util.Initialize(&globalConfig.Market)

	latestId := rds.FindLatestId()
	log.Debugf("latestId:%d", latestId)

	stat(rds, &globalConfig.Item)
	readableAmount(rds)
}

func stat(rds *dao.RdsService, item *config.ItemOption) {
	start := rds.FindLatestId()
	if start == 0 {
		start = item.Start
	} else {
		start += 1
	}
	log.Debugf("start:%d", start)
	for i := item.Start; i<=item.End; i++ {
		single(rds, i, item.DbName)
	}
}

func single(rds *dao.RdsService, id int, dbName string) error {
	fill,err := rds.FindFillById(id)
	if err != nil {
		return nil
	}
	if fill.Fork {
		return nil
	}

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
	setStat(rds, fill.TokenS, symbolS, dbName, amountS.String(), fill.TxHash, fill.ID)

	// tokenB
	tokenB := common.HexToAddress(fill.TokenB)
	symbolB, _ := util.GetSymbolWithAddress(tokenB)
	amountB, _ := new(big.Int).SetString(fill.AmountB, 0)
	splitB, _ := new(big.Int).SetString(fill.SplitB, 0)
	amountB = new(big.Int).Add(amountB, splitB)
	if strings.ToUpper(symbolB) == LRC_SYMBOL {
		amountB = new(big.Int).Add(amountB, lrcTotal)
	}
	setStat(rds, fill.TokenB, symbolB, dbName, amountB.String(), fill.TxHash, fill.ID)

	// lrc
	if strings.ToUpper(symbolS) != LRC_SYMBOL && strings.ToUpper(symbolB) != LRC_SYMBOL {
		setStat(rds, LRC_ADDR, LRC_SYMBOL, dbName, lrcTotal.String(), fill.TxHash, fill.ID)
	}

	return nil
}

func setStat(rds *dao.RdsService, token, symbol, dbName, addAmount, latestTxHash string, latestFillId int) {
	data, err := rds.FindStatDataByToken(token)
	if err != nil {
		data.Amount = addAmount
		data.LatestId = latestFillId
		data.LatestDb = dbName
		data.Token = token
		data.Symbol = symbol
		data.LatestTxHash = latestTxHash
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
		if err := rds.Save(data); err != nil {
			log.Errorf(err.Error())
		} else {
			log.Debugf("update symbol:%s, totalAmount:%s, fillId:%d", symbol,  data.Amount, latestFillId)
		}
	}
}

func readableAmount(rds *dao.RdsService) {
	list, err := rds.GetAllStatData()
	if err != nil {
		log.Fatalf(err.Error())
	}

	for _, v := range list {
		addr := common.HexToAddress(v.Token)
		decimal := big.NewInt(1e18)
		if token, err := util.AddressToToken(addr); err == nil {
			decimal = token.Decimals
		}
		amount, _ := new(big.Int).SetString(v.Amount, 0)
		v.ReadableAmount = new(big.Rat).SetFrac(amount, decimal).FloatString(2)
		if err := rds.Save(v); err != nil {
			log.Errorf(err.Error())
		} else {
			log.Debugf("update symbol:%s, readableAmount:%s", v.Symbol, v.ReadableAmount)
		}
	}
}
