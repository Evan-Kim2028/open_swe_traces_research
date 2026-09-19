/*
Copyright 2023 The ClusterKit Authors.

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

// Package openstackmetadata provides the node-local (metadata-service and
// config-drive based) parts of the OpenStack support, kept separate so that
// nodeup does not link the full cloud provider implementation.
package openstackmetadata

import (
	_ "encoding/json"
	_ "fmt"
	"io"
	_ "net/http"
	_ "os"
	_ "path"
	_ "strings"

	_ "k8s.io/mount-utils"
	utilexec "k8s.io/utils/exec"
)

const (
	// MetadataLatestPath is the path to the metadata on the config drive
	MetadataLatestPath string = "openstack/latest/meta_data.json"

	// MetadataID is the identifier for the metadata service
	MetadataID string = "metadataService"

	// MetadataLastestServiceURL points to the latest metadata of the metadata service
	MetadataLatestServiceURL string = "http://169.254.169.254/" + MetadataLatestPath

	// ConfigDriveID is the identifier for the config drive containing metadata
	ConfigDriveID string = "configDrive"

	// ConfigDriveLabel identifies the config drive by label on the OS
	ConfigDriveLabel string = "config-2"

	// DefaultMetadataSearchOrder defines the default order in which the metadata services are queried
	DefaultMetadataSearchOrder string = ConfigDriveID + ", " + MetadataID

	DiskByLabelPath string = "/dev/disk/by-label/"
)

type Metadata struct {
	// Matches openstack.TagClusterName
	ClusterName string `json:"KubernetesCluster"`
}

type InstanceMetadata struct {
	Name             string    `json:"name"`
	UserMeta         *Metadata `json:"meta"`
	ProjectID        string    `json:"project_id"`
	AvailabilityZone string    `json:"availability_zone"`
	Hostname         string    `json:"hostname"`
	ServerID         string    `json:"uuid"`
}

type MetadataService struct {
	serviceURL      string
	configDrivePath string
	mounter         *mount.SafeFormatAndMount
	mountTarget     string
	searchOrder     string
}

func newMetadataService(serviceURL string, configDrivePath string, mounter *mount.SafeFormatAndMount, mountTarget string, searchOrder string) *MetadataService {
	panic("excised: newMetadataService")
}

// GetLocalMetadata returns a local metadata for the server
func GetLocalMetadata() (*InstanceMetadata, error) {
	panic("excised: GetLocalMetadata")
}

// getFromConfigDrive tries to get metadata by mounting a config drive and returns it as InstanceMetadata
// It will return an error if there is no disk labelled as ConfigDriveLabel or other errors while mounting the disk, or reading the file occur.
func (mds MetadataService) getFromConfigDrive() (*InstanceMetadata, error) {
	panic("excised: MetadataService.getFromConfigDrive")
}

// getFromMetadataService tries to get metadata from a metadata service endpoint and returns it as InstanceMetadata.
// If the service endpoint cannot be contacted or reports a different status than StatusOK it will return an error.
func (mds MetadataService) getFromMetadataService() (*InstanceMetadata, error) {
	panic("excised: MetadataService.getFromMetadataService")
}

// parseMetadata reads JSON data from a Reader and returns it as InstanceMetadata.
func (mds MetadataService) parseMetadata(r io.Reader) (*InstanceMetadata, error) {
	panic("excised: MetadataService.parseMetadata")
}

// getMetadata tries to get metadata for the instance by mounting the config drive and/or querying the metadata service endpoint.
// Depending on the searchOrder it will return data from the first source which successfully returns.
// If all the sources in searchOrder are erroneous it will propagate the last error to its caller.
func (mds MetadataService) getMetadata() (*InstanceMetadata, error) {
	panic("excised: MetadataService.getMetadata")
}

// getDefaultMounter returns a mount and executor interface to use for getting metadata from a config drive
func getDefaultMounter() *mount.SafeFormatAndMount {
	mounter := mount.New("")
	exec := utilexec.New()
	return &mount.SafeFormatAndMount{
		Interface: mounter,
		Exec:      exec,
	}
}
