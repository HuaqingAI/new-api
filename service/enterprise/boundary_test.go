package enterprise_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDepartmentMembershipImplementationAvoidsForbiddenCoreBoundaries(t *testing.T) {
	root := filepath.Clean("../..")
	checkedPaths := []string{
		"constant/enterprise.go",
		"controller/enterprise",
		"dto/enterprise",
		"model/enterprise",
		"router/enterprise-router.go",
		"service/enterprise",
	}
	forbiddenTokens := []string{
		`"github.com/QuantumNous/new-api/relay`,
		`"github.com/QuantumNous/new-api/pkg/billingexpr`,
		`"github.com/QuantumNous/new-api/controller/log`,
		`"github.com/QuantumNous/new-api/model/log`,
		`router/relay-router.go`,
		`docs/openapi/relay.json`,
	}

	for _, checkedPath := range checkedPaths {
		checkedPath := filepath.Join(root, checkedPath)
		info, err := os.Stat(checkedPath)
		require.NoError(t, err)
		if !info.IsDir() {
			content, readErr := os.ReadFile(checkedPath)
			require.NoError(t, readErr)
			for _, token := range forbiddenTokens {
				require.NotContains(t, string(content), token, checkedPath)
			}
			continue
		}
		err = filepath.WalkDir(checkedPath, func(path string, d os.DirEntry, err error) error {
			require.NoError(t, err)
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			content, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			for _, token := range forbiddenTokens {
				require.NotContains(t, string(content), token, path)
			}
			return nil
		})
		require.NoError(t, err)
	}
}

func TestStoryDiffDoesNotModifyForbiddenCoreBoundaries(t *testing.T) {
	root := filepath.Clean("../..")
	forbiddenChangedPaths := []string{
		"relay/",
		"pkg/billingexpr/",
		"model/log.go",
		"controller/log.go",
		"router/relay-router.go",
		"docs/openapi/relay.json",
	}

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = root
	output, err := cmd.Output()
	require.NoError(t, err)

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		path := strings.TrimSpace(line[3:])
		for _, forbidden := range forbiddenChangedPaths {
			require.Falsef(t, strings.HasPrefix(path, forbidden) || path == forbidden, "forbidden Story 1.2 boundary modified: %s", path)
		}
	}
}
