package main

import (
	"fmt"
	"pwdmgr/internal/storage"
	// "pwdmgr/internal/vault"
)

func main() {
	// v := vault.NewVault()

	// entry := vault.NewEntry(
	// 	"GitHub",
	// 	"test",
	// 	"123456",
	// 	"",
	// 	[]string{"work"},
	// )

	// v.AddEntry(entry)

	// storage.SaveVault("vault.dat", []byte("123456"), v)

	v2, err := storage.LoadVault("vault.dat", []byte("123456"))
	if err != nil {
		fmt.Println("加载失败:", err)
		return
	}

	fmt.Println(v2.Entries)
}
