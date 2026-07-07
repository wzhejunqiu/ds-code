package main

import (
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permissionmode"
)

func parsePermissionMode(s string) (permissionmode.Mode, error) {
	return permissionmode.Parse(s)
}

func savePermissionMode(projectRoot string, isProject bool, mode permissionmode.Mode) error {
	return config.SavePermissionMode(projectRoot, isProject, mode)
}
