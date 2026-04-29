package git

const emptyTreeHash = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

func DiffSince(dir, base string) (string, error) {
	if base == "" {
		base = emptyTreeHash
	}
	// NOTE: 未追跡ファイルを diff に載せるため intent-to-add で擬似的にステージする。
	// 実際のオブジェクトは追加されないので user の作業ツリーは変えない。
	_, _ = Run(dir, "add", "-N", "--", ".")
	return Run(dir, "diff", "--no-color", "-U3", base, "--")
}
