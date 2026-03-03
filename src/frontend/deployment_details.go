package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/sirupsen/logrus"
)

var deploymentDetailsMap map[string]string
var log *logrus.Logger

func init() {
	initializeLogger()
	// Use a goroutine to ensure loadDeploymentDetails()'s GCP API
	// calls don't block non-GCP deployments. See issue #685.
	go loadDeploymentDetails()
}

func initializeLogger() {
	log = logrus.New()
	log.Level = logrus.DebugLevel
	log.Formatter = &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339Nano,
	}
	log.Out = os.Stdout
}

// loadDeploymentDetails 采集元数据，包括：
//
//   - HOSTNAME: Pod 的主机名
//   - CLUSTERNAME: Pod 所在的 GKE 集群名称
//   - ZONE: Pod 所在的 GCP 区域和可用区
func loadDeploymentDetails() {
	deploymentDetailsMap = make(map[string]string)
	var metaServerClient = metadata.NewClient(&http.Client{})
	ctx := context.Background()

	podHostname, err := os.Hostname() // 本地环境，通过 os.Hostname() 获取主机名；在 GKE 中，Pod 的主机名默认设置为 Pod 的名字。
	if err != nil {
		log.Error("Failed to fetch the hostname for the Pod", err)
	}

	// 云端环境（GCP）：通过 metadata 客户端访问 Google Cloud 的元数据服务，获取 Pod 所在的集群名称和区域/可用区信息。
	podCluster, err := metaServerClient.InstanceAttributeValueWithContext(ctx, "cluster-name")
	if err != nil {
		log.Error("Failed to fetch the name of the cluster in which the pod is running", err)
	}

	podZone, err := metaServerClient.ZoneWithContext(ctx)
	if err != nil {
		log.Error("Failed to fetch the Zone of the node where the pod is scheduled", err)
	}

	deploymentDetailsMap["HOSTNAME"] = podHostname
	deploymentDetailsMap["CLUSTERNAME"] = podCluster
	deploymentDetailsMap["ZONE"] = podZone

	log.WithFields(logrus.Fields{
		"cluster":  podCluster,
		"zone":     podZone,
		"hostname": podHostname,
	}).Debug("Loaded deployment details")
}
