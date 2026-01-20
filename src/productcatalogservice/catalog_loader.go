package main

import (
	pb "github.com/kznLeaf/curated-store/tree/main/src/productcatalogservice/genproto"
	"encoding/json"
	"os"
)

func loadCatalog(catalog *pb.ListProductsResponse) error {
	catalogMutex.Lock()
	defer catalogMutex.Unlock()

	return loadCatalogFromLocalFile(catalog)
}

// loadCatalogFromLocalFile 从本地文件 products.json 加载产品目录
func loadCatalogFromLocalFile(catalog *pb.ListProductsResponse) error {
	log.Info("正在加载本地文件 products.json")

	data, err := os.ReadFile("products.json")
	if err != nil {
		log.Error("读取 products.json 文件失败:", err)
		return err
	}

	// 把结果读取到 catalog 结构体中
	err = json.Unmarshal(data, catalog)
	if err != nil {
		log.Error("解析 products.json 文件失败:", err)
		return err
	}

	log.Info("成功解析文件products.json")

	return nil
}
