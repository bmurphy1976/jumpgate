package buildinfo

var (
	ReleaseVersion = "unknown"
	Commit         = ""
)

func ServiceVersion() string {
	if Commit == "" {
		return ReleaseVersion
	}
	return ReleaseVersion + "+" + Commit
}
