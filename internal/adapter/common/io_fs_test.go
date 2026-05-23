package common_test

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/fsx"
)

func TestReadIfExistsFSMissingReturnsEmpty(t *testing.T) {
	mem := fsx.Mem()
	got, err := common.ReadIfExistsFS(mem, "/no-such-file.txt")
	require.NoError(t, err)
	assert.Equal(t, "", got)
}

func TestReadIfExistsFSReturnsContent(t *testing.T) {
	mem := fsx.Mem()
	want := "hello from memfs"
	require.NoError(t, afero.WriteFile(mem, "/test.txt", []byte(want), 0o644))

	got, err := common.ReadIfExistsFS(mem, "/test.txt")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestImportMCPFromJSONFileFSHandlesMissingFile(t *testing.T) {
	mem := fsx.Mem()
	servers, err := common.ImportMCPFromJSONFileFS(mem, "/not-here.json", "mcpServers")
	require.NoError(t, err)
	assert.Nil(t, servers)
}

func TestImportMCPFromJSONFileFSExtractsServers(t *testing.T) {
	mem := fsx.Mem()
	body := []byte(`{
		"mcpServers": {
			"myserver": {
				"command": "/usr/bin/mytool",
				"args": ["--flag"],
				"type": "stdio"
			}
		}
	}`)
	require.NoError(t, afero.WriteFile(mem, "/claude.json", body, 0o644))

	servers, err := common.ImportMCPFromJSONFileFS(mem, "/claude.json", "mcpServers")
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.Equal(t, "myserver", servers[0].Name)
	assert.Equal(t, "/usr/bin/mytool", servers[0].Command)
	assert.Equal(t, []string{"--flag"}, servers[0].Args)
	assert.Equal(t, "stdio", servers[0].Transport)
}

func TestImportMCPFromJSONFileFSReadOnlyFS(t *testing.T) {
	// Populate a mem fs then wrap it read-only — reads should still work.
	mem := fsx.Mem()
	body := []byte(`{"mcpServers": {"srv": {"command": "bin"}}}`)
	require.NoError(t, afero.WriteFile(mem, "/config.json", body, 0o644))

	roFS := afero.NewReadOnlyFs(mem)
	servers, err := common.ImportMCPFromJSONFileFS(roFS, "/config.json", "mcpServers")
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.Equal(t, "srv", servers[0].Name)
}
