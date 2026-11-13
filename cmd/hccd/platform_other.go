//go:build !windows

package main

func foregroundExplorerFolder() (string, bool, error) { return "", false, nil }

func registerMenu(register, space bool) error { return nil }
