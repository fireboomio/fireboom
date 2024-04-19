// Package models
/*
 sdk通过git进行管理，并维护GitCommitHash来标识是否有新的版本
*/
package models

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	"go.uber.org/zap"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"

	"go.uber.org/zap/buffer"
)

type invokeAction string

const (
	sdkOnInsert  invokeAction = "onInsert"
	sdkOnUpdate  invokeAction = "onUpdate"
	sdkAfterInit invokeAction = "afterInit"
)

func (s *Sdk) gitClone(action invokeAction) (err error) {
	cloneUrl := ReplaceGithubProxyUrl(s.GitUrl)
	cloneDir := path.Join(consts.RootTemplate, s.Name)
	cloneArgs := []string{"clone", cloneUrl, "-b", s.GitBranch, cloneDir}
	if err = s.executeGitCommand(false, os.Stdout, cloneArgs...); err != nil {
		return
	}

	if len(s.GitCommitHash) == 0 {
		s.GitCommitHash, err = s.fetchCommitHash()
		return
	}

	return s.gitResetAndPull(action)
}

func (s *Sdk) gitResetAndPull(action invokeAction) (err error) {
	if s.isEmptyDir() {
		return s.gitClone(action)
	}

	upgradeRequired := action == sdkOnUpdate
	if !upgradeRequired {
		if action == sdkAfterInit {
			defer func() { err = i18n.NewCustomErrorWithMode(sdkModelName, err, i18n.SdkAlreadyUpToDateError, s.Name) }()
		}

		// 非强制升级需要检查commitHash以确认是否需要升级
		if originCommit, _ := s.fetchCommitHash(); !s.upgradeRequired(originCommit) {
			return
		}

		if err = s.executeGitCommand(true, os.Stdout, "fetch"); err != nil {
			return
		}
	}

	resetArgs := []string{"reset", "--hard"}
	if commit := s.GitCommitHash; len(commit) > 0 {
		resetArgs = append(resetArgs, commit)
	}
	if err = s.executeGitCommand(true, os.Stdout, resetArgs...); err != nil || !upgradeRequired {
		return
	}

	if err = s.executeGitCommand(true, os.Stdout, "pull", "origin", s.GitBranch); err != nil {
		return
	}

	latestCommit, err := s.fetchCommitHash()
	if err != nil {
		return
	}

	// 强制升级完成后比较commitHash确认是否升级成功
	if !s.upgradeRequired(latestCommit) {
		err = i18n.NewCustomErrorWithMode(sdkModelName, nil, i18n.SdkAlreadyUpToDateError, s.Name)
		return
	}

	s.GitCommitHash = latestCommit
	return
}

func (s *Sdk) fetchCommitHash() (commit string, err error) {
	var commitBuf buffer.Buffer
	if err = s.executeGitCommand(true, &commitBuf, "rev-parse", "HEAD"); err != nil {
		return
	}

	commit = strings.TrimSpace(commitBuf.String())
	return
}

func (s *Sdk) upgradeRequired(originCommit string) bool {
	return len(s.GitCommitHash) == 0 || len(originCommit) > 0 && originCommit != s.GitCommitHash
}

func (s *Sdk) isEmptyDir() bool {
	return utils.IsEmptyDirectory(path.Join(consts.RootTemplate, s.Name))
}

func (s *Sdk) executeGitCommand(dirnameRequired bool, writer io.Writer, args ...string) (err error) {
	var stdBuf buffer.Buffer
	cmd := exec.Command("git", args...)
	cmd.Stdout = &stdBuf
	cmd.Stderr = &stdBuf
	if dirnameRequired {
		cmd.Dir = path.Join(consts.RootTemplate, s.Name)
	}
	if err = cmd.Run(); err != nil {
		err = fmt.Errorf("%s(%s)", err.Error(), strings.TrimSpace(stdBuf.String()))
		logger.Warn("exec git command failed", zap.String(sdkModelName, s.Name), zap.String("cmd", cmd.String()), zap.Error(err))
		return
	}

	_, _ = writer.Write(stdBuf.Bytes())
	return
}
