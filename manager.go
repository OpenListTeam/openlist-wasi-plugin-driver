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

type DriverHandle uint32

func (c DriverHandle) GetHandle() uint32 {
	return uint32(c)
}
func (c *DriverHandle) SetHandle(handle uint32) {
	*c = DriverHandle(handle)
}

func (c DriverHandle) LoadConfig(val any) error {
	result := driverimports.LoadConfig(uint32(c))
	if result.IsErr() {
		return errors.New(*result.Err())
	}
	return json.Unmarshal(result.OK().Slice(), val)
}

func (c DriverHandle) SaveConfig(val any) error {
	config, err := json.Marshal(val)
	if err != nil {
		return err
	}
	result := driverimports.SaveConfig(uint32(c), cm.ToList(config))
	if result.IsErr() {
		return errors.New(*result.Err())
	}
	return nil
}

func LoadConfig(driver Driver, val any) error {
	result := driverimports.LoadConfig(driver.GetHandle())
	if result.IsErr() {
		return errors.New(*result.Err())
	}
	return json.Unmarshal(result.OK().Slice(), val)
}

func SaveConfig(driver Driver, val any) error {
	config, err := json.Marshal(val)
	if err != nil {
		return err
	}
	result := driverimports.SaveConfig(driver.GetHandle(), cm.ToList(config))
	if result.IsErr() {
		return errors.New(*result.Err())
	}
	return nil
}
