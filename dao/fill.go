/*

  Copyright 2017 Loopring Project Ltd (Loopring Foundation).

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

*/

package dao

import (
	"github.com/ethereum/go-ethereum/common"
)

type FillEvent struct {
	ID              int    `gorm:"column:id;primary_key;" json:"id"`
	Protocol        string `gorm:"column:contract_address;type:varchar(42)" json:"protocol"`
	DelegateAddress string `gorm:"column:delegate_address;type:varchar(42)" json:"delegateAddress"`
	Owner           string `gorm:"column:owner;type:varchar(42)" json:"owner"`
	RingIndex       int64  `gorm:"column:ring_index;" json:"ringIndex"`
	BlockNumber     int64  `gorm:"column:block_number" json:"blockNumber"`
	CreateTime      int64  `gorm:"column:create_time" json:"createTime"`
	RingHash        string `gorm:"column:ring_hash;varchar(82)" json:"ringHash"`
	FillIndex       int64  `gorm:"column:fill_index" json:"fillIndex"`
	TxHash          string `gorm:"column:tx_hash;type:varchar(82)" json:"txHash"`
	PreOrderHash    string `gorm:"column:pre_order_hash;varchar(82)" json:"preOrderHash"`
	NextOrderHash   string `gorm:"column:next_order_hash;varchar(82)" json:"nextOrderHash"`
	OrderHash       string `gorm:"column:order_hash;type:varchar(82)" json:"orderHash"`
	AmountS         string `gorm:"column:amount_s;type:varchar(40)" json:"amountS"`
	AmountB         string `gorm:"column:amount_b;type:varchar(40)" json:"amountB"`
	TokenS          string `gorm:"column:token_s;type:varchar(42)" json:"tokenS"`
	TokenB          string `gorm:"column:token_b;type:varchar(42)" json:"tokenB"`
	LrcReward       string `gorm:"column:lrc_reward;type:varchar(40)" json:"lrcReward"`
	LrcFee          string `gorm:"column:lrc_fee;type:varchar(40)" json:"lrcFee"`
	SplitS          string `gorm:"column:split_s;type:varchar(40)" json:"splitS"`
	SplitB          string `gorm:"column:split_b;type:varchar(40)" json:"splitB"`
	Market          string `gorm:"column:market;type:varchar(42)" json:"market"`
	LogIndex        int64  `gorm:"column:log_index"`
	Fork            bool   `gorm:"column:fork"`
	Side            string `gorm:"column:side" json:"side"`
	OrderType       string `gorm:"column:order_type" json:"orderType"`
}

func (s *RdsService) FindFillById(id int) (*FillEvent, error) {
	var (
		fill FillEvent
		err  error
	)
	err = s.Db.Where("id = ?", id).First(&fill).Error

	return &fill, err
}

func (s *RdsService) FindFillEvent(txhash string, FillIndex int64) (*FillEvent, error) {
	var (
		fill FillEvent
		err  error
	)
	err = s.Db.Where("tx_hash = ? and fill_index = ?", txhash, FillIndex).Where("fork = ?", false).First(&fill).Error

	return &fill, err
}

func (s *RdsService) FindFillsByRingHash(ringHash common.Hash) ([]FillEvent, error) {
	var (
		fills []FillEvent
		err   error
	)
	err = s.Db.Where("ring_hash = ?", ringHash.Hex()).Where("fork = ?", false).Find(&fills).Error
	return fills, err
}

func (s *RdsService) FillsPageQuery(query map[string]interface{}, pageIndex, pageSize int) (res PageResult, err error) {
	fills := make([]FillEvent, 0)
	res = PageResult{PageIndex: pageIndex, PageSize: pageSize, Data: make([]interface{}, 0)}
	err = s.Db.Where(query).Where("fork=?", false).Order("create_time desc").Offset((pageIndex - 1) * pageSize).Limit(pageSize).Find(&fills).Error
	if err != nil {
		return res, err
	}
	err = s.Db.Model(&FillEvent{}).Where(query).Where("fork=?", false).Count(&res.Total).Error
	if err != nil {
		return res, err
	}

	for _, fill := range fills {
		res.Data = append(res.Data, fill)
	}
	return
}

func (s *RdsService) GetLatestFills(query map[string]interface{}, limit int) (res []FillEvent, err error) {
	fills := make([]FillEvent, 0)
	err = s.Db.Where(query).Where("fork=?", false).Order("create_time desc").Limit(limit).Find(&fills).Error
	if err != nil {
		return res, err
	}
	return fills, nil
}
