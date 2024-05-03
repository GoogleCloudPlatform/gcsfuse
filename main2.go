package main

import (
	control "cloud.google.com/go/storage/control/apiv2"
)

func storageControlClientRetries(sc *control.StorageControlClient) {
	return
}

func CreateGRPCControlClientHandle() (sc *control.StorageControlClient, err error) {

	// Set retries for control client.
	storageControlClientRetries(sc)


	return sc, err
}
