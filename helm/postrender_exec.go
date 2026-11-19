// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"

	"helm.sh/helm/v4/pkg/postrenderer"
)

// execPostRenderer is a PostRenderer implementation that calls a provided binary.
// It replicates the removed helm.sh/helm/v3/pkg/postrender.NewExec functionality
// for Helm v4, where post-renderers are normally plugins.
type execPostRenderer struct {
	binaryPath string
	args       []string
}

// NewExecPostRenderer returns a PostRenderer implementation that calls the provided binary.
// It returns an error if the binary cannot be found. If the path does not
// contain any separators, it will search in $PATH, otherwise it will resolve
// any relative paths to a fully qualified path.
func NewExecPostRenderer(binaryPath string, args ...string) (postrenderer.PostRenderer, error) {
	fullPath, err := exec.LookPath(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("unable to find binary at %s: %w", binaryPath, err)
	}
	abs, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, err
	}
	return &execPostRenderer{abs, args}, nil
}

// Run executes the configured binary for the post render step.
func (p *execPostRenderer) Run(renderedManifests *bytes.Buffer) (*bytes.Buffer, error) {
	cmd := exec.Command(p.binaryPath, p.args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	postRendered := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = postRendered
	cmd.Stderr = stderr

	go func() {
		defer stdin.Close()
		_, _ = io.Copy(stdin, renderedManifests)
	}()
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error while running command %s. error output:\n%s: %w", p.binaryPath, stderr.String(), err)
	}

	return postRendered, nil
}
