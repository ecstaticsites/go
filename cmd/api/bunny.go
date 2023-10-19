package api

type BunnyClient struct {}

func (b BunnyClient) CreateStorageZone() error {
	return nil
}

func (b BunnyClient) CreatePullZone() error {
	return nil
}

func (b BunnyClient) ConfigureLogsForPullZone() error {
	return nil
}
