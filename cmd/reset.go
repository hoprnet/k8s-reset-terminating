/*
Copyright (c) 2020 Jian Zhang
Licensed under MIT https://github.com/jianz/jianz.github.io/blob/master/LICENSE
*/

package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	rook "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/spf13/cobra"
	"go.etcd.io/etcd/clientv3"
)

var (
	etcdCA, etcdCert, etcdKey, etcdHost string
	etcdPort                            int

	k8sKeyPrefix      string = "registry"
	resourceGroupName string = "ceph.rook.io"
	resourceType      string
	resourceName      string
	serializerPrefix  = []byte{0x6b, 0x38, 0x73, 0x00}

	cmd = &cobra.Command{
		Use:   "k8s_reset [flags] <resource name>",
		Short: "Reset the Terminating resource back to previous status.",
		Long:  "Reset the Terminating resource back to previous status.\nPlease visit https://github.com/hoprnet/k8s-reset-terminating for the detailed explanation.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("requires one resource name argument")
			}
			resourceName = args[0]
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			err := resetResource()
			return err
		},
	}
)

// Execute reset the Terminating PersistentVolume to Bound status.
func Execute() {
	cmd.Flags().StringVar(&etcdCA, "etcd-ca", "ca.crt", "CA Certificate used by etcd")
	cmd.Flags().StringVar(&etcdCert, "etcd-cert", "etcd.crt", "Public key used by etcd")
	cmd.Flags().StringVar(&etcdKey, "etcd-key", "etcd.key", "Private key used by etcd")
	cmd.Flags().StringVar(&etcdHost, "etcd-host", "localhost", "The etcd domain name or IP")
	cmd.Flags().StringVar(&resourceType, "k8s-resource-type", "cephfilesystems", "The plural lower case name of the resource type.")
	cmd.Flags().IntVar(&etcdPort, "etcd-port", 2379, "The etcd port number")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func resetResource() error {
	etcdCli, err := etcdClient()
	if err != nil {
		return fmt.Errorf("cannot connect to etcd: %v", err)
	}
	defer etcdCli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return recoverResource(ctx, etcdCli)
}

func etcdClient() (*clientv3.Client, error) {
	ca, err := ioutil.ReadFile(etcdCA)
	if err != nil {
		return nil, err
	}

	keyPair, err := tls.LoadX509KeyPair(etcdCert, etcdKey)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(ca)

	return clientv3.New(clientv3.Config{
		Endpoints:   []string{fmt.Sprintf("%s:%d", etcdHost, etcdPort)},
		DialTimeout: 2 * time.Second,
		TLS: &tls.Config{
			RootCAs:      certPool,
			Certificates: []tls.Certificate{keyPair},
		},
	})
}

func recoverResource(ctx context.Context, client *clientv3.Client) error {
	var key = fmt.Sprintf("/%s/%s/%s/%s", k8sKeyPrefix, resourceGroupName, resourceType, resourceName)
	fmt.Println("Searching for key", key)
	resp, err := client.Get(ctx, key)
	if err != nil {
		return err
	}

	if len(resp.Kvs) < 1 {
		return fmt.Errorf("cannot find resource [%s] in etcd with key [%s]\nplease check the k8s-key-prefix and the resource name are set correctly", resourceName, key)
	}

	fmt.Println("Previous:", string(resp.Kvs[0].Value))
	var new_data []byte
	switch resourceType {
	case "cephfilesystems":
		var resource rook.CephFilesystem
		err = json.Unmarshal(resp.Kvs[0].Value, &resource)
		if err != nil {
			return err
		}
		if resource.ObjectMeta.DeletionTimestamp == nil {
			return fmt.Errorf("resource [%s] is not in terminating status", resourceName)
		}
		resource.ObjectMeta.DeletionTimestamp = nil
		resource.ObjectMeta.DeletionGracePeriodSeconds = nil
		new_data, err = json.Marshal(&resource)
	case "cephobjectstores":
		var resource rook.CephObjectStore
		err = json.Unmarshal(resp.Kvs[0].Value, &resource)
		if err != nil {
			return err
		}
		if resource.ObjectMeta.DeletionTimestamp == nil {
			return fmt.Errorf("resource [%s] is not in terminating status", resourceName)
		}
		resource.ObjectMeta.DeletionTimestamp = nil
		resource.ObjectMeta.DeletionGracePeriodSeconds = nil
		new_data, err = json.Marshal(&resource)
	case "cephclusters":
		var resource rook.CephCluster
		err = json.Unmarshal(resp.Kvs[0].Value, &resource)
		if err != nil {
			return err
		}
		if resource.ObjectMeta.DeletionTimestamp == nil {
			return fmt.Errorf("resource [%s] is not in terminating status", resourceName)
		}
		resource.ObjectMeta.DeletionTimestamp = nil
		resource.ObjectMeta.DeletionGracePeriodSeconds = nil
		new_data, err = json.Marshal(&resource)
	default:
		return errors.New("Ceph Type not supported")
	}

	resp.Kvs[0].Value = new_data

	fmt.Println("After:", string(resp.Kvs[0].Value))

	// Write the updated protobuf value back to etcd
	client.Put(ctx, key, string(resp.Kvs[0].Value))
	return nil
}
