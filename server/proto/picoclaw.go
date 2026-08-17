package proto

type GetPicoclawSourceRsp struct {
	DownloadUrl string `json:"downloadUrl"`
	ChecksumUrl string `json:"checksumUrl"`
	// What is used when no override is set.
	DefaultDownloadUrl string `json:"defaultDownloadUrl"`
	DefaultChecksumUrl string `json:"defaultChecksumUrl"`
}

type SetPicoclawSourceReq struct {
	DownloadUrl string `json:"downloadUrl" form:"downloadUrl"`
	ChecksumUrl string `json:"checksumUrl" form:"checksumUrl"`
}
