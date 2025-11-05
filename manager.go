package openlistwasiplugindriver

import (
	"context"
	"encoding/json"
	"errors"

	driverimports "github.com/OpenListTeam/openlist-wasi-plugin-driver/binding/openlist/plugin-driver/host"
	drivertypes "github.com/OpenListTeam/openlist-wasi-plugin-driver/binding/openlist/plugin-driver/types"

	"go.bytecodealliance.org/cm"
)

type RootID struct {
	RootFolderID string `json:"root_folder_id"`
}

func (r *RootID) GetRoot(ctx context.Context) (*drivertypes.Object, error) {
	return &drivertypes.Object{
		ID:       r.RootFolderID,
		Name:     "root",
		IsFolder: true,
	}, nil
}

type RootPath struct {
	RootFolderPath string `json:"root_folder_path"`
}

func (r *RootPath) GetRoot(ctx context.Context) (*drivertypes.Object, error) {
	return &drivertypes.Object{
		Path:     r.RootFolderPath,
		Name:     "root",
		IsFolder: true,
	}, nil
}

type DriverHandle struct {
}

func (c DriverHandle) GetHandle() uint32 {
	return hostHeadle
}

func (c DriverHandle) LoadConfig(val any) error {
	return LoadConfig(val)
}

func (c DriverHandle) SaveConfig(val any) error {
	return SaveConfig(val)
}

var hostHeadle uint32 = 0

func LoadConfig(val any) error {
	result := driverimports.LoadConfig(hostHeadle)
	if result.IsErr() {
		return errors.New(*result.Err())
	}
	return json.Unmarshal(result.OK().Slice(), val)
}

func SaveConfig(val any) error {
	config, err := json.Marshal(val)
	if err != nil {
		return err
	}
	result := driverimports.SaveConfig(hostHeadle, cm.ToList(config))
	if result.IsErr() {
		return errors.New(*result.Err())
	}
	return nil
}
