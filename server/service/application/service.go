package application

const (
	// Updates come from this fork's releases, not Sipeed's CDN. Pointing these
	// upstream would offer a stock build as an "update" and quietly replace
	// everything this fork adds, so both channels resolve here. There is no
	// separate preview channel, hence the same URL for both.
	//
	// GitHub's /releases/latest/ skips prereleases, so a release has to be
	// published or promoted as stable before a device will see it.
	StableURL  = "https://github.com/ddcash/NanoKVM-Extended/releases/latest/download"
	PreviewURL = "https://github.com/ddcash/NanoKVM-Extended/releases/latest/download"

	AppDir    = "/kvmapp"
	BackupDir = "/root/old"
	CacheDir  = "/root/.kvmcache"

	updateWorkspacePrefix = "nanokvm-update-"
	cacheDirMode          = 0o700
	maxPackageSize        = uint64(1 << 30)
	maxExpandedSize       = uint64(2 << 30)
	maxArchiveEntries     = 100_000
	minFreeReserve        = uint64(128 << 20)
	freeReservePercent    = uint64(5)
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}
