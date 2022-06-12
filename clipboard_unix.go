// Copyright 2013 @atotto. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build freebsd || linux || netbsd || openbsd || solaris || dragonfly
// +build freebsd linux netbsd openbsd solaris dragonfly

package clipboard

import (
	"errors"
	"os"
	"os/exec"
)

const (
	xsel               = "xsel"
	xclip              = "xclip"
	powershell         = "powershell.exe"
	winclip            = "clip.exe"
	wlcopy             = "wl-copy"
	wlpaste            = "wl-paste"
	termuxClipboardGet = "termux-clipboard-get"
	termuxClipboardSet = "termux-clipboard-set"
)

var (
	pasteCmdArgs map[string][]string
	copyCmdArgs  map[string][]string

	trimDos bool

	xselPasteArgs = map[string][]string{
		"primary":   []string{xsel, "--output"},
		"clipboard": []string{xsel, "--output", "--clipboard"},
	}
	xselCopyArgs = map[string][]string{
		"primary":   []string{xsel, "--input"},
		"clipboard": []string{xsel, "--input", "--clipboard"},
	}

	xclipPasteArgs = map[string][]string{
		"primary":   []string{xclip, "-out"},
		"clipboard": []string{xclip, "-out", "-selection", "clipboard"},
	}
	xclipCopyArgs = map[string][]string{
		"primary":   []string{xclip, "-in"},
		"clipboard": []string{xclip, "-in", "-selection", "clipboard"},
	}

	wlpasteArgs = map[string][]string{
		"primary":   []string{wlpaste, "--no-newline", "--primary"},
		"clipboard": []string{wlpaste, "--no-newline"},
	}
	wlcopyArgs = map[string][]string{
		"primary":   []string{wlcopy, "--primary"},
		"clipboard": []string{wlcopy},
	}

	termuxPasteArgs = map[string][]string{
		"primary":   []string{termuxClipboardGet},
		"clipboard": []string{termuxClipboardGet},
	}
	termuxCopyArgs = map[string][]string{
		"primary":   []string{termuxClipboardSet},
		"clipboard": []string{termuxClipboardSet},
	}

	powershellPasteArgs = map[string][]string{
		"primary":   []string{powershell, "Get-Clipboard"},
		"clipboard": []string{powershell, "Get-Clipboard"},
	}
	winclipCopyArgs = map[string][]string{
		"primary":   []string{winclip},
		"clipboard": []string{winclip},
	}

	missingCommands = errors.New("No clipboard utilities available. Please install xsel, xclip, wl-clipboard or Termux:API add-on for termux-clipboard-get/set.")

	internalClipboards map[string]string
)

// verify install makes sure that the given exec is installed
// and operating correctly. In some cases, xclip/xsel can be installed
// but not working because an X server is not running.
func verifyInstall(cmd string) bool {
	if _, err := exec.LookPath(cmd); err == nil {
		// first check if to read from the clipboard
		// if this is possible, then xclip/xsel is working properly
		pasteCmd := getPasteCommand("primary")
		_, err := pasteCmd.Output()
		if err == nil {
			return true
		}

		// If the previous command didn't work, this could be because the clipboard
		// has no contents. Now we will try to copy an empty string into the clipboard.
		copyCmd := getCopyCommand("primary")
		in, err := copyCmd.StdinPipe()
		if err != nil {
			return false
		}

		if err := copyCmd.Start(); err != nil {
			return false
		}
		if _, err := in.Write([]byte{}); err != nil {
			return false
		}
		if err := in.Close(); err != nil {
			return false
		}
		return copyCmd.Wait() == nil
	}

	// if exec was not found in path we know it is not installed
	return false
}

func initialize() error {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		pasteCmdArgs = wlpasteArgs
		copyCmdArgs = wlcopyArgs

		if _, err := exec.LookPath(wlcopy); err == nil {
			if _, err := exec.LookPath(wlpaste); err == nil {
				return nil
			}
		}
	}

	pasteCmdArgs = xclipPasteArgs
	copyCmdArgs = xclipCopyArgs

	if verifyInstall(xclip) {
		return nil
	}

	pasteCmdArgs = xselPasteArgs
	copyCmdArgs = xselCopyArgs

	if verifyInstall(xsel) {
		return nil
	}

	pasteCmdArgs = termuxPasteArgs
	copyCmdArgs = termuxCopyArgs

	if _, err := exec.LookPath(termuxClipboardSet); err == nil {
		if _, err := exec.LookPath(termuxClipboardGet); err == nil {
			return nil
		}
	}

	pasteCmdArgs = powershellPasteArgs
	copyCmdArgs = winclipCopyArgs

	if _, err := exec.LookPath(winclip); err == nil {
		if _, err := exec.LookPath(powershell); err == nil {
			trimDos = true
			return nil
		}
	}

	if internalClipboards == nil {
		internalClipboards = make(map[string]string)
	}
	Unsupported = true

	return errors.New("System clipboard not found, install xclip/xsel")
}

func getPasteCommand(register string) *exec.Cmd {
	return exec.Command(pasteCmdArgs[register][0], pasteCmdArgs[register][1:]...)
}

func getCopyCommand(register string) *exec.Cmd {
	return exec.Command(copyCmdArgs[register][0], copyCmdArgs[register][1:]...)
}

func readAll(register string) (string, error) {
	if Unsupported {
		if text, ok := internalClipboards[register]; ok {
			return text, nil
		}
		return "", nil
	}
	pasteCmd := getPasteCommand(register)
	out, err := pasteCmd.Output()
	if err != nil {
		return "", err
	}
	result := string(out)
	if trimDos && len(result) > 1 {
		result = result[:len(result)-2]
	}
	return result, nil
}

func writeAll(text, register string) error {
	if Unsupported {
		internalClipboards[register] = text
		return nil
	}
	copyCmd := getCopyCommand(register)
	in, err := copyCmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := copyCmd.Start(); err != nil {
		return err
	}
	if _, err := in.Write([]byte(text)); err != nil {
		return err
	}
	if err := in.Close(); err != nil {
		return err
	}
	return copyCmd.Wait()
}
