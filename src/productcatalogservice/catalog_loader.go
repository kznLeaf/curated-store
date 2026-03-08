package main

import (
	"os"

	pb "github.com/kznLeaf/curated-store/src/productcatalogservice/genproto"
	"google.golang.org/protobuf/encoding/protojson"
)

func loadCatalog(catalog *pb.ListProductsResponse) error {
	catalogMutex.Lock()
	defer catalogMutex.Unlock()

	return loadCatalogFromLocalFile(catalog)
}

// loadCatalogFromLocalFile 从本地文件 products.json 加载产品目录
func loadCatalogFromLocalFile(catalog *pb.ListProductsResponse) error {
	log.Info("loading local file products.json")

	data, err := os.ReadFile("products.json")
	if err != nil {
		log.Error("failed to read products.json:", err)
		return err
	}

	// 把结果读取到 catalog 结构体中
	if err = protojson.Unmarshal(data, catalog); err != nil {
		log.Error("failed to parse products.json:", err)
		return err
	}

	log.Info("successfully parsed products.json")

	return nil
}
