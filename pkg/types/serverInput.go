package types

type ScanPostReq struct {
	AssetName       string
	AssetRemoteUrls []string
	Branch          string
	Revision        string
	Findings        []UnifiedFinding
	FailedDetectors []string
}

type DevicePostReq struct {
	DeviceName string
}

type DependencycheckScanPostReq struct {
	ManifestFiles []FileReq
}

type FileReq struct {
	Path    string
	Content []byte
}

type FixedFileContentPostReq struct {
	FileContent       string
	ProblemLineNumber int
	Hint              string
}
