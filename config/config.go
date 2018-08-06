package config

import (
	"errors"
	"os"
	"reflect"

	"github.com/Loopring/relay-lib/dao"
	"github.com/naoina/toml"
	util "github.com/Loopring/relay-lib/marketutil"
	"go.uber.org/zap"
	"github.com/Loopring/relay-lib/cache/redis"
)

func LoadConfig() *GlobalConfig {
	file := "/Users/fukun/projects/gohome/src/github.com/dylenfu/fill-stat/config/debug.toml"

	io, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer io.Close()

	c := &GlobalConfig{}
	if err := toml.NewDecoder(io).Decode(c); err != nil {
		panic(err)
	}
	return c
}

type GlobalConfig struct {
	Log              zap.Config
	Mysql            dao.MysqlOptions
	Redis            redis.RedisOptions
	Market           util.MarketOptions
	Item 			 ItemOption
}

type ItemOption struct {
	DbName string
	Start int
	End int
}

func Validator(cv reflect.Value) (bool, error) {
	for i := 0; i < cv.NumField(); i++ {
		cvt := cv.Type().Field(i)

		if cv.Field(i).Type().Kind() == reflect.Struct {
			if res, err := Validator(cv.Field(i)); nil != err {
				return res, err
			}
		} else {
			if "true" == cvt.Tag.Get("required") {
				if !isSet(cv.Field(i)) {
					return false, errors.New("The field " + cvt.Name + " in config must be setted")
				}
			}
		}
	}

	return true, nil
}

func isSet(v reflect.Value) bool {
	switch v.Type().Kind() {
	case reflect.Invalid:
		return false
	case reflect.String:
		return v.String() != ""
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() != 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() != 0
	case reflect.Map:
		return len(v.MapKeys()) != 0
	}
	return true
}

