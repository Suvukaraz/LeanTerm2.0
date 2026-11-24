package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PureEdit (a second process) changes only the editor settings and keeps what
// PureTerm wrote to config.toml after PureEdit loaded it.
func TestUpdateOnDiskKeepsOtherChanges(t *testing.T) {
	dir := t.TempDir()
	pureTerm, err := NewStore(StoreOptions{ConfigDir: dir, NoBackup: true})
	require.NoError(t, err)
	require.NoError(t, pureTerm.Load())
	require.NoError(t, pureTerm.Save())

	pureEdit, err := NewStore(StoreOptions{ConfigDir: dir, NoBackup: true})
	require.NoError(t, err)
	require.NoError(t, pureEdit.Load())

	// PureTerm saves another setting after PureEdit started.
	pureTerm.Config().Terminal.ScreenshotsFolder = "/tmp/shots"
	require.NoError(t, pureTerm.Save())

	require.NoError(t, pureEdit.UpdateOnDisk(func(cfg *Config) { cfg.Editor.TabWidth = 8 }))
	assert.Equal(t, 8, pureEdit.Config().Editor.TabWidth)

	data, err := os.ReadFile(pureTerm.ConfigPath())
	require.NoError(t, err)
	reloaded, err := pureTerm.parseConfig(data)
	require.NoError(t, err)
	assert.Equal(t, 8, reloaded.Editor.TabWidth)
	assert.Equal(t, "/tmp/shots", reloaded.Terminal.ScreenshotsFolder)

	// Invalid values are rejected and nothing is written.
	assert.Error(t, pureEdit.UpdateOnDisk(func(cfg *Config) { cfg.Editor.TabWidth = 99 }))
	data2, _ := os.ReadFile(pureTerm.ConfigPath())
	assert.Equal(t, data, data2)
}
