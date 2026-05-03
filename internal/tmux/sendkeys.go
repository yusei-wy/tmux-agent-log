package tmux

import (
	"encoding/base64"
	"io"
	"os"
	"os/exec"
)

type SendResultKind int

const (
	SendResultOK SendResultKind = iota
	SendResultFallbackClipboard
	SendResultFailed
)

type SendResult struct {
	Kind SendResultKind
	Err  error
}

func SendToPane(paneID, text string) SendResult {
	return sendToPaneWithWriters("", paneID, text, os.Stdout, os.Stderr)
}

// TODO: 第 5 引数の stderr writer は未配線。エラーハンドリング見直し時に send-keys 失敗時の出力先として使う。
func sendToPaneWithWriters(socket, paneID, text string, clipboard io.Writer, _ io.Writer) SendResult {
	exists, err := paneExistsWithSocket(socket, paneID)
	if err != nil {
		return SendResult{Kind: SendResultFailed, Err: err}
	}
	if !exists {
		seq := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\x07"
		if _, err := io.WriteString(clipboard, seq); err != nil {
			return SendResult{Kind: SendResultFailed, Err: err}
		}
		return SendResult{Kind: SendResultFallbackClipboard}
	}

	if err := runTmux(socket, "send-keys", "-t", paneID, "-l", text); err != nil {
		return SendResult{Kind: SendResultFailed, Err: err}
	}
	if err := runTmux(socket, "send-keys", "-t", paneID, "Enter"); err != nil {
		return SendResult{Kind: SendResultFailed, Err: err}
	}
	return SendResult{Kind: SendResultOK}
}

func runTmux(socket string, args ...string) error {
	full := []string{}
	if socket != "" {
		full = append(full, "-S", socket)
	}
	full = append(full, args...)
	//nolint:gosec // tmux の引数はラッパー内で組み立てる variable な配列。設計上の意図。
	return exec.Command("tmux", full...).Run()
}
