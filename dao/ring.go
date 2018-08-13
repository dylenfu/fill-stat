package dao

type RingMinedEvent struct {
	ID                 int    `gorm:"column:id;primary_key" json:"id"`
	Protocol           string `gorm:"column:contract_address;type:varchar(42)" json:"protocol"`
	DelegateAddress    string `gorm:"column:delegate_address;type:varchar(42)" json:"delegateAddress"`
	RingIndex          string `gorm:"column:ring_index;type:varchar(40)" json:"ringIndex"`
	RingHash           string `gorm:"column:ring_hash;type:varchar(82)" json:"ringHash"`
	TxHash             string `gorm:"column:tx_hash;type:varchar(82)" json:"txHash"`
	OrderHashList      string `gorm:"column:order_hash_list;type:text"`
	Miner              string `gorm:"column:miner;type:varchar(42);" json:"miner"`
	FeeRecipient       string `gorm:"column:fee_recipient;type:varchar(42)" json:"feeRecipient"`
	IsRinghashReserved bool   `gorm:"column:is_ring_hash_reserved;" json:"isRinghashReserved"`
	BlockNumber        int64  `gorm:"column:block_number;type:bigint" json:"blockNumber"`
	TotalLrcFee        string `gorm:"column:total_lrc_fee;type:varchar(40)" json:"totalLrcFee"`
	TradeAmount        int    `gorm:"column:trade_amount" json:"tradeAmount"`
	Time               int64  `gorm:"column:time;type:bigint" json:"timestamp"`
	Status             uint8  `gorm:"column:status;type:tinyint(4)"`
	Fork               bool   `gorm:"column:fork"`
	GasLimit           string `gorm:"column:gas_limit;type:varchar(50)"`
	GasUsed            string `gorm:"column:gas_used;type:varchar(50)"`
	GasPrice           string `gorm:"column:gas_price;type:varchar(50)"`
	Err                string `gorm:"column:err;type:text" json:"err"`
}

func (s *RdsService) FindRingMinedById(id int) (*RingMinedEvent, error) {
	var (
		model RingMinedEvent
		err   error
	)

	err = s.Db.Where("id=?", id).Where("status=?", 2).Where("fork = ?", false).First(&model).Error

	return &model, err
}

func (s *RdsService) FindRingMined(txhash string) (*RingMinedEvent, error) {
	var (
		model RingMinedEvent
		err   error
	)

	err = s.Db.Where("tx_hash=?", txhash).Where("fork = ?", false).First(&model).Error

	return &model, err
}
