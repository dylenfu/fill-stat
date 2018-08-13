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

type StatData struct {
	ID              int    `gorm:"column:id;primary_key;"`
	Token 			string 	`gorm:"column:token;type:varchar(42)"`
	Symbol          string `gorm:"column:symbol;type:varchar(42)"`
	Amount         	string `gorm:"column:amount;type:varchar(40)"`
	TxCount 		int    `gorm:"column:fill_number"`
	ReadableAmount  string `gorm:"column:readable_amount;type:varchar(40)"`
	LatestTxHash 	string `gorm:"column:latest_tx_hash;type:varchar(82)"`
	LatestDb 		string `gorm:"column:latest_db;type:varchar(40)"`
	LatestId 		int    `gorm:"column:latest_id"`
}

func (s *RdsService) FindStatDataByToken(token string) (*StatData, error) {
	var (
		sd   StatData
		err  error
	)
	err = s.Db.Where("token = ?", token).First(&sd).Error

	return &sd, err
}

func (s *RdsService) FindLatestId(dbName string) int {
	var (
		sd   StatData
		err  error
	)
	err = s.Db.Where("latest_db=?", dbName).Where("id > ?", 0).Order("latest_id DESC").First(&sd).Error

	if err != nil {
		return 0
	}
	return sd.LatestId
}

func (s *RdsService) UpdateAmount(token,totalAmount string) error {
	return s.Db.Model(&StatData{}).Where("token", token).Update("amount", totalAmount).Error
}

func (s *RdsService) GetAllStatData() ([]StatData, error) {
	var (
		list []StatData
		err error
	)

	err = s.Db.Where("id > ?", 0).Find(&list).Error
	return list, err
}
