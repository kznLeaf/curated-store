package main

import (
	"encoding/json"
	"os"
)

// 这个文件用于定义获取汇率表的逻辑。暂时使用从本地文件加载汇率表的方式，后期会改成周期性查询货币汇率。

// loadCurrencyData 辅助函数，读取本地的汇率表，返回map[string]float64，键是货币代码，值是相对于CNY的汇率。
// TODO 以后周期性更新汇率表
func loadCurrencyData() (map[string]float64, error) {
	currencyMutex.Lock()
	defer currencyMutex.Unlock()

	data, err := os.ReadFile("currency_conversion.json")
	if err != nil {
		return nil, err
	}
	var rates map[string]float64
	err = json.Unmarshal(data, &rates)
	return rates, err
}
